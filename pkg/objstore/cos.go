//go:build cos

package objstore

import (
	"context"
	"errors"
	"io"
	"time"
)

// COS 是腾讯云对象存储驱动的占位实现。
//
// 接入步骤：
//  1. go get github.com/tencentyun/cos-go-sdk-v5
//  2. 在 Put/Get/Delete/Stat 内调用 SDK；
//  3. SignOneShot 推荐做法：服务端先 PutObject 到私有 Bucket，再用本地 HMAC token 表（参考 Local）
//     将一次性 token 与真实 object key 关联；用户 GET 命中后再 302 到 cos 临时签名 URL。
//  4. 构建：go build -tags=cos ./...
type COS struct {
	Bucket    string
	Region    string
	SecretID  string
	SecretKey string
}

func NewCOS(bucket, region, id, key string) (*COS, error) {
	if bucket == "" || region == "" || id == "" || key == "" {
		return nil, errors.New("objstore.cos: missing credentials")
	}
	return &COS{Bucket: bucket, Region: region, SecretID: id, SecretKey: key}, nil
}

func (c *COS) Put(ctx context.Context, key string, r io.Reader, contentType string) (ObjectMeta, error) {
	return ObjectMeta{}, errors.New("objstore.cos: TODO Put")
}
func (c *COS) Get(ctx context.Context, key string) (io.ReadCloser, ObjectMeta, error) {
	return nil, ObjectMeta{}, errors.New("objstore.cos: TODO Get")
}
func (c *COS) Delete(ctx context.Context, key string) error {
	return errors.New("objstore.cos: TODO Delete")
}
func (c *COS) Stat(ctx context.Context, key string) (ObjectMeta, bool, error) {
	return ObjectMeta{}, false, errors.New("objstore.cos: TODO Stat")
}
func (c *COS) SignOneShot(ctx context.Context, key string, ttl time.Duration) (string, error) {
	return "", errors.New("objstore.cos: TODO SignOneShot")
}
func (c *COS) ResolveOneShot(ctx context.Context, token string) (string, error) {
	return "", errors.New("objstore.cos: TODO ResolveOneShot")
}
