package aicweb

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

var blockedKeys = map[string]struct{}{
	"password":            {},
	"oldPassword":         {},
	"newPassword":         {},
	"token":               {},
	"accessToken":         {},
	"refreshToken":        {},
	"turnstileToken":      {},
	"cfTurnstileResponse": {},
	"captcha":             {},
}

func isBlocked(key string) bool {
	k := strings.ToLower(key)
	if _, ok := blockedKeys[key]; ok {
		return true
	}
	_, ok := blockedKeys[k]
	return ok
}

// sanitizeValue 递归删除敏感键
func sanitizeValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k := range x {
			if isBlocked(k) {
				delete(x, k)
				continue
			}
			x[k] = sanitizeValue(x[k])
		}
		return x
	case []any:
		for i := range x {
			x[i] = sanitizeValue(x[i])
		}
		return x
	default:
		return v
	}
}

// SanitizeRawJSON 读取原始 JSON 并返回“剔除敏感字段后”的 RawMessage
func SanitizeRawJSON(r io.Reader, maxBytes int64) (json.RawMessage, error) {
	if maxBytes <= 0 {
		maxBytes = 1 << 20 // 1MB 缺省上限
	}
	lr := io.LimitReader(r, maxBytes+1)
	body, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, errors.New("payload too large")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	m = sanitizeValue(m).(map[string]any)
	out, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}
