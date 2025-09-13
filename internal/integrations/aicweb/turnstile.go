package aicweb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

const turnstileEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type TurnstileVerifier interface {
	Verify(ctx context.Context, token, remoteIP string) (ok bool, errs []string, err error)
	Enabled() bool
}

type httpTurnstile struct {
	client *http.Client
	secret string
	enable bool
}

func NewTurnstileFromEnv() TurnstileVerifier {
	secret := os.Getenv("TURNSTILE_SECRET")
	enable := os.Getenv("TURNSTILE_ENABLED")
	// 默认开启；当 env 显式为 "false" 时关闭
	enabled := secret != "" && enable != "false"
	return &httpTurnstile{
		client: &http.Client{Timeout: 5 * time.Second},
		secret: secret,
		enable: enabled,
	}
}

func (h *httpTurnstile) Enabled() bool { return h.enable }

func (h *httpTurnstile) Verify(ctx context.Context, token, remoteIP string) (bool, []string, error) {
	// 允许在 dev 关闭校验
	if !h.enable {
		return true, nil, nil
	}
	if token == "" {
		return false, []string{"missing-token"}, nil
	}

	form := url.Values{}
	form.Set("secret", h.secret)
	form.Set("response", token)
	if ip := net.ParseIP(remoteIP); ip != nil {
		form.Set("remoteip", remoteIP)
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, turnstileEndpoint, bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.client.Do(req)
	if err != nil {
		return false, nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Success     bool     `json:"success"`
		ErrorCodes  []string `json:"error-codes"`
		Hostname    string   `json:"hostname"`
		ChallengeTS string   `json:"challenge_ts"`
		Action      string   `json:"action"`
		CData       string   `json:"cdata"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, nil, err
	}
	// Cloudflare 会返回 200 + success=false 的语义错误
	if !out.Success {
		return false, out.ErrorCodes, nil
	}
	return true, nil, nil
}

// 从请求中抽取 Turnstile token（优先 Header，兼容 Body 字段）
func getTurnstileToken(c RequestCtx) (string, error) {
	// 约定：前端把 token 放到 Header: CF-Turnstile-Response
	if v := c.GetHeader("CF-Turnstile-Response"); v != "" {
		return v, nil
	}
	// 兼容 body: { "turnstileToken": "..." } 或 { "cfTurnstileResponse": "..." }
	var tmp struct {
		TurnstileToken      string `json:"turnstileToken"`
		CFTurnstileResponse string `json:"cfTurnstileResponse"`
	}
	if err := c.ShouldBindBodyWithJSON(&tmp); err == nil {
		if tmp.TurnstileToken != "" {
			return tmp.TurnstileToken, nil
		}
		if tmp.CFTurnstileResponse != "" {
			return tmp.CFTurnstileResponse, nil
		}
	}
	return "", errors.New("turnstile token not found")
}

// 让 handler 无需直接依赖 gin，做个极薄接口便于单测
type RequestCtx interface {
	GetHeader(string) string
	ClientIP() string
	ShouldBindBodyWithJSON(any) error
}
