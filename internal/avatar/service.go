package avatar

import (
	"bytes"
	"crypto/md5" // #nosec G401: acceptable for content addressing
	"encoding/hex"
	"errors"
	"github.com/chai2010/webp"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var ErrTooLarge = errors.New("avatar: file too large")

func IsTooLarge(err error) bool { return errors.Is(err, ErrTooLarge) }

type Service struct {
	Dir       string  // 保存目录
	URLPrefix string  // 返回 URL 的前缀
	MaxBytes  int64   // 单文件最大字节
	Quality   float32 // webp 质量 0-100
}

func NewServiceFromEnv() (*Service, error) {
	dir := getenv("AVATAR_DIR", "assets/avatar")
	urlp := getenv("AVATAR_URL_PREFIX", "/assets/avatar")
	maxMB, _ := strconv.Atoi(getenv("AVATAR_MAX_MB", "5"))
	q, _ := strconv.Atoi(getenv("AVATAR_WEBP_QUALITY", "80"))

	// 确保目录存在
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	log.Printf("[avatar] dir=%q url_prefix=%q maxMB=%d quality=%d", dir, urlp, maxMB, q)

	return &Service{
		Dir:       dir,
		URLPrefix: urlp,
		MaxBytes:  int64(maxMB) * (1 << 20),
		Quality:   float32(q),
	}, nil
}

func getenv(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

// ProcessAndStore: 读取 r（受限大小）-> 解码 -> 编码 WebP -> md5 命名 -> 落盘
func (s *Service) ProcessAndStore(r io.Reader) (avatarID, filePath, url string, err error) {
	// 读取并限制体积
	var src bytes.Buffer
	if s.MaxBytes > 0 {
		if _, err = io.CopyN(&src, r, s.MaxBytes+1); err != nil && err != io.EOF {
			return "", "", "", err
		}
		if int64(src.Len()) > s.MaxBytes {
			return "", "", "", ErrTooLarge
		}
	} else {
		if _, err = io.Copy(&src, r); err != nil {
			return "", "", "", err
		}
	}

	// 尝试通用解码（jpeg/png/gif）
	img, _, decErr := image.Decode(bytes.NewReader(src.Bytes()))
	if decErr != nil {
		// 再尝试 WebP 解码
		if img2, err2 := webp.Decode(bytes.NewReader(src.Bytes())); err2 == nil {
			img = img2
		} else {
			return "", "", "", decErr
		}
	}

	// 编码为 WebP
	var out bytes.Buffer
	opt := &webp.Options{Lossless: false, Quality: s.Quality}
	if err = webp.Encode(&out, img, opt); err != nil {
		return "", "", "", err
	}

	// MD5 作为文件名 & avatarID（基于“编码后的字节”）
	sum := md5.Sum(out.Bytes()) // #nosec G401
	avatarID = hex.EncodeToString(sum[:])
	filename := avatarID + ".webp"
	filePath = filepath.Join(s.Dir, filename)

	// 若文件不存在则写入（幂等）
	if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
		if err = os.WriteFile(filePath, out.Bytes(), 0o644); err != nil {
			return "", "", "", err
		}
	}

	// 生成 URL（简单拼接）
	url = strings.TrimRight(s.URLPrefix, "/") + "/" + filename
	return avatarID, filePath, url, nil
}
