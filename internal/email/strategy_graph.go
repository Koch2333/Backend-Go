package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// 通过 REST 调用：
// 1) POST https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token （client_credentials）
// 2) POST https://graph.microsoft.com/v1.0/users/{fromUPN}/sendMail
type graphSender struct {
	tenant     string
	clientID   string
	secret     string
	fromUPN    string
	httpClient *http.Client
}

func newGraphSenderFromEnv() Sender {
	tenant := strings.TrimSpace(os.Getenv("GRAPH_TENANT_ID"))
	clientID := strings.TrimSpace(os.Getenv("GRAPH_CLIENT_ID"))
	secret := strings.TrimSpace(os.Getenv("GRAPH_CLIENT_SECRET"))
	from := strings.TrimSpace(os.Getenv("GRAPH_FROM_UPN"))
	if tenant == "" || clientID == "" || secret == "" || from == "" {
		return nil
	}
	return &graphSender{
		tenant:     tenant,
		clientID:   clientID,
		secret:     secret,
		fromUPN:    from,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (g *graphSender) Name() string { return "graph" }

func (g *graphSender) Send(to, subject, htmlBody, textBody string) error {
	if g == nil || g.httpClient == nil {
		return nil
	}

	// 1) 取 access_token（client_credentials）
	token, err := g.fetchToken(context.Background())
	if err != nil {
		return fmt.Errorf("graph: get token failed: %w", err)
	}

	// 2) 组装 sendMail payload
	var contentType, content string
	if strings.TrimSpace(htmlBody) != "" {
		contentType, content = "HTML", htmlBody
	} else {
		contentType, content = "Text", textBody
	}

	payload := map[string]interface{}{
		"message": map[string]interface{}{
			"subject": subject,
			"body": map[string]interface{}{
				"contentType": contentType,
				"content":     content,
			},
			"toRecipients": []map[string]interface{}{
				{"emailAddress": map[string]interface{}{"address": to}},
			},
		},
		"saveToSentItems": true,
	}
	b, _ := json.Marshal(payload)

	// 3) 调 Graph 发送
	sendURL := "https://graph.microsoft.com/v1.0/users/" + url.PathEscape(g.fromUPN) + "/sendMail"
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, sendURL, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("graph: sendMail http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("graph: sendMail status=%d, body=%s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

// fetchToken 使用 client_credentials 换 access_token（不依赖任何外部包）
func (g *graphSender) fetchToken(ctx context.Context) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", g.clientID)
	form.Set("client_secret", g.secret)
	form.Set("scope", "https://graph.microsoft.com/.default")

	tokenURL := "https://login.microsoftonline.com/" + url.PathEscape(g.tenant) + "/oauth2/v2.0/token"
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	type tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	var tr tokenResp
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("decode token resp: %w", err)
	}
	if tr.AccessToken == "" {
		if tr.Error != "" {
			return "", fmt.Errorf("token error: %s (%s)", tr.Error, tr.ErrorDesc)
		}
		return "", fmt.Errorf("empty access_token, status=%d", resp.StatusCode)
	}
	return tr.AccessToken, nil
}
