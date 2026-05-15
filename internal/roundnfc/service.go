package roundnfc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"backend-go/internal/auth/adminpw"
	"backend-go/internal/authflow"
	"backend-go/internal/risk"
	"backend-go/pkg/objstore"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
)

var (
	ErrTooLarge         = errors.New("roundnfc: file too large")
	ErrUnsupportedMedia = errors.New("roundnfc: unsupported media type")
)

type Config struct {
	DBPath            string
	ObjectDir         string
	ObjectHMACKey     []byte
	ObjectTTL         time.Duration
	MaxUploadBytes    int64
	TurnstileSecret   string
	RateLimitPerMin   int
	AdminUsername     string
	AdminPasswordHash string
	JWTSecret         []byte
	JWTTTL            time.Duration
	TOTPIssuer        string
	WebAuthnRPID      string
	WebAuthnRPName    string
	WebAuthnOrigins   []string
}

func ConfigFromEnv() Config {
	atoiOr := func(env string, def int) int {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
		return def
	}
	getStr := func(env, def string) string {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return v
		}
		return def
	}
	origins := strings.Split(getStr("ROUNDNFC_WEBAUTHN_ORIGINS", "http://localhost:5174,http://localhost:8081"), ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}
	return Config{
		DBPath:            getStr("ROUNDNFC_SQLITE_PATH", "databases/roundnfc/roundnfc.db"),
		ObjectDir:         getStr("ROUNDNFC_OBJECT_DIR", "storage/roundnfc/objects"),
		ObjectHMACKey:     []byte(os.Getenv("ROUNDNFC_OBJECT_HMAC_KEY")),
		ObjectTTL:         time.Duration(atoiOr("ROUNDNFC_OBJECT_TTL_SECONDS", 120)) * time.Second,
		MaxUploadBytes:    int64(atoiOr("ROUNDNFC_MAX_UPLOAD_MB", 8)) * (1 << 20),
		TurnstileSecret:   os.Getenv("ROUNDNFC_TURNSTILE_SECRET"),
		RateLimitPerMin:   atoiOr("ROUNDNFC_RATELIMIT_PER_MIN", 12),
		AdminUsername:     getStr("ROUNDNFC_ADMIN_USERNAME", "admin"),
		AdminPasswordHash: adminpw.Resolve("roundnfc", "ROUNDNFC"),
		JWTSecret:         []byte(os.Getenv("ROUNDNFC_JWT_SECRET")),
		JWTTTL:            time.Duration(atoiOr("ROUNDNFC_JWT_TTL_HOURS", 12)) * time.Hour,
		TOTPIssuer:        getStr("ROUNDNFC_TOTP_ISSUER", "RoundNFC"),
		WebAuthnRPID:      getStr("ROUNDNFC_WEBAUTHN_RPID", "localhost"),
		WebAuthnRPName:    getStr("ROUNDNFC_WEBAUTHN_RP_NAME", "RoundNFC Admin"),
		WebAuthnOrigins:   origins,
	}
}

type Service struct {
	cfg     Config
	store   *Store
	objects objstore.Storage
	rl      *risk.RateLimiter
}

func NewServiceFromEnv() (*Service, error) {
	cfg := ConfigFromEnv()
	store, err := openStore(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	if len(cfg.ObjectHMACKey) < 16 {
		return nil, errors.New("ROUNDNFC_OBJECT_HMAC_KEY must be set (>=16 bytes)")
	}
	local, err := objstore.NewLocal(cfg.ObjectDir, cfg.ObjectHMACKey)
	if err != nil {
		return nil, err
	}
	return &Service{
		cfg:     cfg,
		store:   store,
		objects: local,
		rl:      risk.NewRateLimiter(cfg.RateLimitPerMin, time.Minute),
	}, nil
}

func (s *Service) Close() error { return s.store.Close() }

// AuthFlowConfig returns the authflow.Config for this service.
func (s *Service) AuthFlowConfig() authflow.Config {
	return authflow.Config{
		Store:             s.store,
		AdminUsername:     s.cfg.AdminUsername,
		AdminPasswordHash: s.cfg.AdminPasswordHash,
		JWTSecret:         s.cfg.JWTSecret,
		JWTTTL:            s.cfg.JWTTTL,
		TOTPIssuer:        s.cfg.TOTPIssuer,
		WebAuthnRPID:      s.cfg.WebAuthnRPID,
		WebAuthnRPName:    s.cfg.WebAuthnRPName,
		WebAuthnOrigins:   s.cfg.WebAuthnOrigins,
	}
}

// allowedImageMIME 仅允许常见位图格式。
var allowedImageMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// IngestImage 受限大小读取 + MIME 嗅探 + 按 sha256 内容寻址并落盘。
func (s *Service) IngestImage(ctx context.Context, prefix string, r io.Reader) (key, mime string, size int64, err error) {
	var buf bytes.Buffer
	if s.cfg.MaxUploadBytes > 0 {
		n, copyErr := io.CopyN(&buf, r, s.cfg.MaxUploadBytes+1)
		if copyErr != nil && copyErr != io.EOF {
			return "", "", 0, copyErr
		}
		if n > s.cfg.MaxUploadBytes {
			return "", "", 0, ErrTooLarge
		}
	} else {
		if _, err := io.Copy(&buf, r); err != nil {
			return "", "", 0, err
		}
	}
	mt := mimetype.Detect(buf.Bytes())
	mimeStr := mt.String()
	ext, ok := allowedImageMIME[mimeStr]
	if !ok {
		return "", "", 0, ErrUnsupportedMedia
	}
	sum := sha256.Sum256(buf.Bytes())
	hexsum := hex.EncodeToString(sum[:])
	objectKey := path.Join(prefix, hexsum[:2], hexsum+ext)
	if _, exists, _ := s.objects.Stat(ctx, objectKey); !exists {
		if _, err := s.objects.Put(ctx, objectKey, &buf, mimeStr); err != nil {
			return "", "", 0, err
		}
	}
	return objectKey, mimeStr, int64(buf.Len()), nil
}

func (s *Service) SignObject(ctx context.Context, key string) (string, error) {
	return s.objects.SignOneShot(ctx, key, s.cfg.ObjectTTL)
}

func (s *Service) ResolveObject(ctx context.Context, token string) (io.ReadCloser, objstore.ObjectMeta, error) {
	key, err := s.objects.ResolveOneShot(ctx, token)
	if err != nil {
		return nil, objstore.ObjectMeta{}, err
	}
	return s.objects.Get(ctx, key)
}

// PublicBadge 将内部 imageUrl（可能是 object key）转换为一次性下载 URL。
func (s *Service) PublicBadge(ctx context.Context, b *Badge, urlPrefix string) Badge {
	out := *b
	if out.ImageURL != "" && !isAbsoluteURL(out.ImageURL) {
		if token, err := s.SignObject(ctx, out.ImageURL); err == nil {
			out.ImageURL = strings.TrimRight(urlPrefix, "/") + "/objects/" + token
		} else {
			out.ImageURL = ""
		}
	}
	return out
}

func isAbsoluteURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func newID(prefix string) string { return prefix + "_" + uuid.NewString() }

func hashIP(ip, salt string) string {
	if ip == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(salt + "|" + ip))
	return hex.EncodeToString(sum[:8])
}
