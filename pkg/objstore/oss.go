//go:build oss

package objstore

import (
	"context"
	"errors"
	"io"
	"time"
)

// OSS 是阿里云对象存储驱动的占位实现。
//
// 接入步骤：
//  1. go get github.com/aliyun/aliyun-oss-go-sdk
//  2. 实现 Put/Get/Delete/Stat；
//  3. SignOneShot 走「私有 bucket + 我方一次性 token」组合，详见 cos.go 注释。
//  4. 构建：go build -tags=oss ./...
type OSS struct {
	Bucket    string
	Endpoint  string
	SecretID  string
	SecretKey string
}

func NewOSS(bucket, endpoint, id, key string) (*OSS, error) {
	if bucket == "" || endpoint == "" || id == "" || key == "" {
		return nil, errors.New("objstore.oss: missing credentials")
	}
	return &OSS{Bucket: bucket, Endpoint: endpoint, SecretID: id, SecretKey: key}, nil
}

func (o *OSS) Put(ctx context.Context, key string, r io.Reader, contentType string) (ObjectMeta, error) {
	return ObjectMeta{}, errors.New("objstore.oss: TODO Put")
}
func (o *OSS) Get(ctx context.Context, key string) (io.ReadCloser, ObjectMeta, error) {
	return nil, ObjectMeta{}, errors.New("objstore.oss: TODO Get")
}
func (o *OSS) Delete(ctx context.Context, key string) error {
	return errors.New("objstore.oss: TODO Delete")
}
func (o *OSS) Stat(ctx context.Context, key string) (ObjectMeta, bool, error) {
	return ObjectMeta{}, false, errors.New("objstore.oss: TODO Stat")
}
func (o *OSS) SignOneShot(ctx context.Context, key string, ttl time.Duration) (string, error) {
	return "", errors.New("objstore.oss: TODO SignOneShot")
}
func (o *OSS) ResolveOneShot(ctx context.Context, token string) (string, error) {
	return "", errors.New("objstore.oss: TODO ResolveOneShot")
}
