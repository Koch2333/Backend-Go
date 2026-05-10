package risk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type TurnstileResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// VerifyTurnstile 调用 Cloudflare Turnstile siteverify。
// 当 secret 为空（开发态）时直接放行。
func VerifyTurnstile(ctx context.Context, secret, token, remoteIP string) (bool, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return true, nil
	}
	if token == "" {
		return false, nil
	}
	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://challenges.cloudflare.com/turnstile/v0/siteverify",
		strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var r TurnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return false, err
	}
	return r.Success, nil
}

func TurnstileSecretFromEnv() string {
	return strings.TrimSpace(os.Getenv("ROUNDNFC_TURNSTILE_SECRET"))
}
