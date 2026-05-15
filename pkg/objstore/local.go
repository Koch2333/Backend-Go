package objstore

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Local struct {
	Dir     string
	HMACKey []byte

	mu      sync.Mutex
	pending map[string]int64
}

func NewLocal(dir string, hmacKey []byte) (*Local, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if len(hmacKey) < 16 {
		return nil, errors.New("objstore.local: hmac key must be at least 16 bytes")
	}
	l := &Local{Dir: dir, HMACKey: hmacKey, pending: map[string]int64{}}
	go l.gcLoop()
	return l, nil
}

func (l *Local) abs(key string) string {
	clean := filepath.Clean("/" + strings.ReplaceAll(key, "..", ""))
	return filepath.Join(l.Dir, strings.TrimPrefix(clean, "/"))
}

func (l *Local) Put(_ context.Context, key string, r io.Reader, contentType string) (ObjectMeta, error) {
	dst := l.abs(key)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return ObjectMeta{}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".tmp-*")
	if err != nil {
		return ObjectMeta{}, err
	}
	n, copyErr := io.Copy(tmp, r)
	if cerr := tmp.Close(); copyErr == nil {
		copyErr = cerr
	}
	if copyErr != nil {
		_ = os.Remove(tmp.Name())
		return ObjectMeta{}, copyErr
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		_ = os.Remove(tmp.Name())
		return ObjectMeta{}, err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return ObjectMeta{Key: key, ContentType: contentType, Size: n}, nil
}

func (l *Local) Get(_ context.Context, key string) (io.ReadCloser, ObjectMeta, error) {
	f, err := os.Open(l.abs(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ObjectMeta{}, ErrNotFound
		}
		return nil, ObjectMeta{}, err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, ObjectMeta{}, err
	}
	ct := contentTypeByExt(filepath.Ext(key))
	return f, ObjectMeta{Key: key, ContentType: ct, Size: st.Size()}, nil
}

func (l *Local) Delete(_ context.Context, key string) error {
	if err := os.Remove(l.abs(key)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (l *Local) Stat(_ context.Context, key string) (ObjectMeta, bool, error) {
	st, err := os.Stat(l.abs(key))
	if err != nil {
		if os.IsNotExist(err) {
			return ObjectMeta{}, false, nil
		}
		return ObjectMeta{}, false, err
	}
	return ObjectMeta{Key: key, Size: st.Size()}, true, nil
}

// token = base64url(payload) + "." + base64url(hmac); payload = key|exp|nonce
func (l *Local) SignOneShot(_ context.Context, key string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	nb := make([]byte, 16)
	if _, err := rand.Read(nb); err != nil {
		return "", err
	}
	nonce := hex.EncodeToString(nb)
	exp := time.Now().Add(ttl).Unix()
	payload := fmt.Sprintf("%s|%d|%s", key, exp, nonce)

	mac := hmac.New(sha256.New, l.HMACKey)
	mac.Write([]byte(payload))
	tok := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." +
		base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	l.mu.Lock()
	l.pending[nonce] = exp
	l.mu.Unlock()
	return tok, nil
}

func (l *Local) ResolveOneShot(_ context.Context, token string) (string, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", ErrTokenInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrTokenInvalid
	}
	gotSig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrTokenInvalid
	}
	mac := hmac.New(sha256.New, l.HMACKey)
	mac.Write(payload)
	if !hmac.Equal(gotSig, mac.Sum(nil)) {
		return "", ErrTokenInvalid
	}
	segs := strings.SplitN(string(payload), "|", 3)
	if len(segs) != 3 {
		return "", ErrTokenInvalid
	}
	key, expStr, nonce := segs[0], segs[1], segs[2]
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return "", ErrTokenInvalid
	}
	if time.Now().Unix() > exp {
		return "", ErrTokenExpired
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	storedExp, ok := l.pending[nonce]
	if !ok {
		return "", ErrTokenConsumed
	}
	if storedExp != exp {
		return "", ErrTokenInvalid
	}
	delete(l.pending, nonce)
	return key, nil
}

func (l *Local) gcLoop() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for range t.C {
		now := time.Now().Unix()
		l.mu.Lock()
		for k, v := range l.pending {
			if v < now {
				delete(l.pending, k)
			}
		}
		l.mu.Unlock()
	}
}

func contentTypeByExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".webp":
		return "image/webp"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	}
	return "application/octet-stream"
}
