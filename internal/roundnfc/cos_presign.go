package roundnfc

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const cosPresignTTL = 5 * time.Minute

var ErrCOSNotConfigured = errors.New("roundnfc: cos not configured")

type UploadPresign struct {
	UploadURL string `json:"uploadUrl"`
	ObjectKey string `json:"objectKey"`
	Method    string `json:"method"`
	ExpiresIn int    `json:"expiresIn"`
}

func (s *Service) PresignUpload(ctx context.Context, badgeID, fileName, contentType, purpose string) (UploadPresign, error) {
	_ = ctx
	key, err := buildUploadObjectKey(badgeID, fileName, contentType, purpose)
	if err != nil {
		return UploadPresign{}, err
	}
	u, err := s.presignCOSPut(key, cosPresignTTL)
	if err != nil {
		return UploadPresign{}, err
	}
	return UploadPresign{
		UploadURL: u,
		ObjectKey: key,
		Method:    "PUT",
		ExpiresIn: int(cosPresignTTL / time.Second),
	}, nil
}

func buildUploadObjectKey(badgeID, fileName, contentType, purpose string) (string, error) {
	badgeID = safePathSegment(badgeID)
	if badgeID == "" {
		return "", errors.New("badgeId required")
	}
	ext, err := uploadExt(fileName, contentType, purpose)
	if err != nil {
		return "", err
	}
	return path.Join("roundnfc", "nfc-writes", badgeID, uuid.NewString()+ext), nil
}

func uploadExt(fileName, contentType, purpose string) (string, error) {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	p := strings.ToLower(strings.TrimSpace(purpose))
	if p == "" || p == "nfc-write" || p == "nfc-writes" || p == "download" || p == "user-download" {
		if ct == "" {
			ct = "image/jpeg"
		}
		if ct != "image/jpeg" && ct != "image/jpg" {
			return "", errors.New("nfc write photos must be image/jpeg")
		}
		return ".jpg", nil
	}
	switch ct {
	case "image/jpeg", "image/jpg":
		return ".jpg", nil
	case "image/webp":
		return ".webp", nil
	case "image/png":
		return ".png", nil
	}
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".jpg", ".jpeg":
		return ".jpg", nil
	case ".webp":
		return ".webp", nil
	case ".png":
		return ".png", nil
	}
	return "", errors.New("unsupported contentType")
}

var unsafePathChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func safePathSegment(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, `/\`)
	return unsafePathChars.ReplaceAllString(v, "_")
}

func (s *Service) presignCOSPut(objectKey string, ttl time.Duration) (string, error) {
	cfg := s.cfg
	if cfg.COSBucket == "" || cfg.COSRegion == "" || cfg.COSSecretID == "" || cfg.COSSecretKey == "" {
		return "", ErrCOSNotConfigured
	}
	scheme := strings.ToLower(strings.TrimSpace(cfg.COSScheme))
	if scheme != "http" {
		scheme = "https"
	}
	now := time.Now().Unix()
	exp := now + int64(ttl/time.Second)
	keyTime := fmt.Sprintf("%d;%d", now, exp)
	host := fmt.Sprintf("%s.cos.%s.myqcloud.com", cfg.COSBucket, cfg.COSRegion)
	uri := "/" + strings.TrimLeft(objectKey, "/")

	headerList := "host"
	httpString := strings.Join([]string{
		"put",
		uri,
		"",
		"host=" + strings.ToLower(host) + "\n",
	}, "\n")
	httpHash := sha1Hex([]byte(httpString))
	stringToSign := strings.Join([]string{"sha1", keyTime, httpHash, ""}, "\n")
	signKey := hmacSHA1Hex([]byte(cfg.COSSecretKey), []byte(keyTime))
	signature := hmacSHA1Hex([]byte(signKey), []byte(stringToSign))

	q := url.Values{}
	q.Set("q-sign-algorithm", "sha1")
	q.Set("q-ak", cfg.COSSecretID)
	q.Set("q-sign-time", keyTime)
	q.Set("q-key-time", keyTime)
	q.Set("q-header-list", headerList)
	q.Set("q-url-param-list", "")
	q.Set("q-signature", signature)

	return (&url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     uri,
		RawQuery: q.Encode(),
	}).String(), nil
}

func sha1Hex(b []byte) string {
	h := sha1.Sum(b)
	return hex.EncodeToString(h[:])
}

func hmacSHA1Hex(key, msg []byte) string {
	mac := hmac.New(sha1.New, key)
	mac.Write(msg)
	return hex.EncodeToString(mac.Sum(nil))
}
