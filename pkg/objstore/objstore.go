// Package objstore 是对象存储抽象。
// 现有实现：Local（本地磁盘 + HMAC 一次性 URL）。
// 预留实现：COS / OSS（构建 tag 控制，见 cos.go / oss.go）。
package objstore

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	ErrNotFound      = errors.New("objstore: not found")
	ErrTokenInvalid  = errors.New("objstore: token invalid")
	ErrTokenExpired  = errors.New("objstore: token expired")
	ErrTokenConsumed = errors.New("objstore: token already consumed")
)

type ObjectMeta struct {
	Key         string
	ContentType string
	Size        int64
	ETag        string
}

type Storage interface {
	Put(ctx context.Context, key string, r io.Reader, contentType string) (ObjectMeta, error)
	Get(ctx context.Context, key string) (io.ReadCloser, ObjectMeta, error)
	Delete(ctx context.Context, key string) error
	Stat(ctx context.Context, key string) (ObjectMeta, bool, error)

	// SignOneShot 颁发一次性 token；只在第一次 ResolveOneShot 时有效。
	SignOneShot(ctx context.Context, key string, ttl time.Duration) (token string, err error)
	// ResolveOneShot 校验并消费 token，返回真实 key。
	ResolveOneShot(ctx context.Context, token string) (key string, err error)
}
