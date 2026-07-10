package roundnfc

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const cosPresignTTL = 5 * time.Minute
const cosDownloadPresignTTL = 2 * time.Minute

var ErrCOSNotConfigured = errors.New("roundnfc: cos not configured")

type UploadPresign struct {
	UploadURL string            `json:"uploadUrl"`
	ObjectKey string            `json:"objectKey"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresIn int               `json:"expiresIn"`
}

type COSObjectPresign struct {
	URL       string `json:"url"`
	ObjectKey string `json:"objectKey"`
	ExpiresIn int    `json:"expiresIn"`
}

func (s *Service) PresignUpload(ctx context.Context, badgeID, fileName, contentType, purpose string) (UploadPresign, error) {
	_ = ctx
	key, err := buildUploadObjectKey(badgeID, fileName, contentType, purpose)
	if err != nil {
		return UploadPresign{}, err
	}
	u, headers, err := s.presignCOSPut(key, cosPresignTTL)
	if err != nil {
		return UploadPresign{}, err
	}
	log.Printf("roundnfc cos presign badge_id=%s purpose=%s object_key=%s bucket=%s region=%s secret_id_prefix=%s expires_in=%ds", badgeID, purpose, key, s.cfg.COSBucket, s.cfg.COSRegion, shortSecretID(s.cfg.COSSecretID), int(cosPresignTTL/time.Second))
	return UploadPresign{
		UploadURL: u,
		ObjectKey: key,
		Method:    "PUT",
		Headers:   headers,
		ExpiresIn: int(cosPresignTTL / time.Second),
	}, nil
}

func (s *Service) PresignCOSObject(ctx context.Context, objectKey, urlPrefix string) (COSObjectPresign, error) {
	key := strings.TrimSpace(objectKey)
	if err := validateAppCOSObjectKey(key); err != nil {
		return COSObjectPresign{}, err
	}
	token, err := s.SignCOSObject(ctx, key)
	if err != nil {
		return COSObjectPresign{}, err
	}
	return COSObjectPresign{
		URL:       strings.TrimRight(urlPrefix, "/") + "/cos-objects/" + token,
		ObjectKey: key,
		ExpiresIn: int(s.cfg.ObjectTTL / time.Second),
	}, nil
}

func validateAppCOSObjectKey(key string) error {
	if key == "" {
		return errors.New("objectKey required")
	}
	if isAbsoluteURL(key) || strings.HasPrefix(key, "/") || strings.Contains(key, "..") {
		return errors.New("invalid objectKey")
	}
	if strings.HasPrefix(key, "roundnfc/coser-photos/") || strings.HasPrefix(key, "roundnfc/nfc-writes/") {
		return nil
	}
	return errors.New("objectKey is not an app upload")
}

func shortSecretID(v string) string {
	v = strings.TrimSpace(v)
	if len(v) <= 6 {
		return v
	}
	return v[:6] + "..."
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
	p := strings.ToLower(strings.TrimSpace(purpose))
	if p == "coser-photo" || p == "cn-photo" || p == "badge-coser" {
		return path.Join("roundnfc", "coser-photos", badgeID, uuid.NewString()+ext), nil
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

func (s *Service) presignCOSPut(objectKey string, ttl time.Duration) (string, map[string]string, error) {
	cfg := s.cfg
	if cfg.COSBucket == "" || cfg.COSRegion == "" || cfg.COSSecretID == "" || cfg.COSSecretKey == "" {
		return "", nil, ErrCOSNotConfigured
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

	uploadURL := (&url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   uri,
	}).String()
	headers := map[string]string{
		"Authorization": strings.Join([]string{
			"q-sign-algorithm=sha1",
			"q-ak=" + cfg.COSSecretID,
			"q-sign-time=" + keyTime,
			"q-key-time=" + keyTime,
			"q-header-list=" + headerList,
			"q-url-param-list=",
			"q-signature=" + signature,
		}, "&"),
	}
	return uploadURL, headers, nil
}

func (s *Service) presignCOSGet(objectKey string, ttl time.Duration) (string, error) {
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
		"get",
		uri,
		"",
		"host=" + strings.ToLower(host) + "\n",
	}, "\n")
	httpHash := sha1Hex([]byte(httpString))
	stringToSign := strings.Join([]string{"sha1", keyTime, httpHash, ""}, "\n")
	signKey := hmacSHA1Hex([]byte(cfg.COSSecretKey), []byte(keyTime))
	signature := hmacSHA1Hex([]byte(signKey), []byte(stringToSign))

	u := &url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   uri,
	}
	q := u.Query()
	q.Set("q-sign-algorithm", "sha1")
	q.Set("q-ak", cfg.COSSecretID)
	q.Set("q-sign-time", keyTime)
	q.Set("q-key-time", keyTime)
	q.Set("q-header-list", headerList)
	q.Set("q-url-param-list", "")
	q.Set("q-signature", signature)
	u.RawQuery = q.Encode()
	return u.String(), nil
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
