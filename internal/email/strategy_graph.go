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

// ---- Cloud selector ----
type graphCloudEnv struct {
	tokenBase string // AAD token base
	graphBase string // Graph API base
	scope     string // OAuth scope
	name      string // for logging
}

func pickCloud() graphCloudEnv {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GRAPH_CLOUD"))) {
	case "cn", "china", "21vianet":
		return graphCloudEnv{
			tokenBase: "https://login.chinacloudapi.cn",
			graphBase: "https://microsoftgraph.chinacloudapi.cn",
			scope:     "https://microsoftgraph.chinacloudapi.cn/.default",
			name:      "cn",
		}
	default:
		return graphCloudEnv{
			tokenBase: "https://login.microsoftonline.com",
			graphBase: "https://graph.microsoft.com",
			scope:     "https://graph.microsoft.com/.default",
			name:      "global",
		}
	}
}

// 通过 REST 调用：
// 1) POST {tokenBase}/{tenant}/oauth2/v2.0/token（client_credentials）
// 2) POST {graphBase}/v1.0/users/{from}/sendMail
type graphSender struct {
	tenant     string
	clientID   string
	secret     string
	fromUPN    string // 可选
	fromID     string // 优先
	httpClient *http.Client
}

func newGraphSenderFromEnv() Sender {
	tenant := strings.TrimSpace(os.Getenv("GRAPH_TENANT_ID"))
	clientID := strings.TrimSpace(os.Getenv("GRAPH_CLIENT_ID"))
	secret := strings.TrimSpace(os.Getenv("GRAPH_CLIENT_SECRET"))
	fromUPN := strings.TrimSpace(os.Getenv("GRAPH_FROM_UPN"))
	fromID := strings.TrimSpace(os.Getenv("GRAPH_FROM_ID"))

	if tenant == "" || clientID == "" || secret == "" || (fromUPN == "" && fromID == "") {
		// 关键信息缺失则返回 nil，让工厂回退到 smtp/none
		return nil
	}

	// 小日志（可按需保留/去掉）
	fmt.Printf("[email/graph] cloud=%s tenant=%s fromID=%s fromUPN=%s\n",
		pickCloud().name, short(tenant), short(fromID), fromUPN)

	return &graphSender{
		tenant:     tenant,
		clientID:   clientID,
		secret:     secret,
		fromUPN:    fromUPN,
		fromID:     fromID,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (g *graphSender) Name() string { return "graph" }

func (g *graphSender) Send(to, subject, htmlBody, textBody string) error {
	if g == nil {
		return fmt.Errorf("graph: sender is nil")
	}
	if g.httpClient == nil {
		return fmt.Errorf("graph: httpClient is nil")
	}
	env := pickCloud()

	// 1) token
	tok, err := g.fetchToken(context.Background(), env)
	if err != nil {
		fmt.Printf("[email/graph] token error: %v\n", err)
		return err
	}

	// 2) payload
	var contentType, content string
	if strings.TrimSpace(htmlBody) != "" {
		contentType, content = "HTML", htmlBody
	} else {
		contentType, content = "Text", textBody
	}
	payload := map[string]any{
		"message": map[string]any{
			"subject": subject,
			"body":    map[string]any{"contentType": contentType, "content": content},
			"toRecipients": []map[string]any{
				{"emailAddress": map[string]any{"address": to}},
			},
		},
		"saveToSentItems": true,
	}
	b, _ := json.Marshal(payload)

	// 3) send
	var userPath string
	if g.fromID != "" {
		userPath = url.PathEscape(g.fromID) // /users/{id}
	} else {
		userPath = url.PathEscape(g.fromUPN) // /users/{upn}
	}
	sendURL := env.graphBase + "/v1.0/users/" + userPath + "/sendMail"

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, sendURL, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		fmt.Printf("[email/graph] http error: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("[email/graph] sendMail status=%d body=%s\n", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("graph: sendMail status=%d", resp.StatusCode)
	}

	fmt.Printf("[email/graph] sent ok to=%s subject=%q\n", to, subject)
	return nil
}

// fetchToken 使用 client_credentials 换 access_token（不依赖任何外部包）
func (g *graphSender) fetchToken(ctx context.Context, env graphCloudEnv) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", g.clientID)
	form.Set("client_secret", g.secret)
	form.Set("scope", env.scope)

	tokenURL := env.tokenBase + "/" + url.PathEscape(g.tenant) + "/oauth2/v2.0/token"
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

// ---- helpers ----
func short(s string) string {
	if len(s) > 8 {
		return s[:8] + "..."
	}
	return s
}
