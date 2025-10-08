package msconsent

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Config struct {
	TenantID    string
	ClientID    string
	RedirectURI string
}

func loadConfig() Config {
	return Config{
		TenantID:    strings.TrimSpace(os.Getenv("GRAPH_TENANT_ID")),
		ClientID:    strings.TrimSpace(os.Getenv("GRAPH_CLIENT_ID")),
		RedirectURI: strings.TrimSpace(os.Getenv("GRAPH_ADMIN_CONSENT_REDIRECT_URI")),
	}
}

type stateStore struct {
	mu sync.Mutex
	m  map[string]time.Time
}

var states = &stateStore{m: map[string]time.Time{}}

func (s *stateStore) New(ttl time.Duration) (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	st := base64.RawURLEncoding.EncodeToString(b)
	s.mu.Lock()
	s.m[st] = time.Now().Add(ttl)
	s.mu.Unlock()
	return st, nil
}

func (s *stateStore) VerifyAndDelete(st string) bool {
	now := time.Now()
	s.mu.Lock()
	exp, ok := s.m[st]
	if ok {
		delete(s.m, st)
	}
	s.mu.Unlock()
	return ok && now.Before(exp)
}

func adminConsentURL(tenant, clientID, redirectURI, state string) (string, error) {
	if clientID == "" || redirectURI == "" {
		return "", fmt.Errorf("missing clientID or redirectURI")
	}
	base := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0/adminconsent",
		url.PathEscape(firstNonEmpty(tenant, "organizations")))
	u, _ := url.Parse(base)
	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func firstNonEmpty(v string, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

// GET {prefix}/admin-consent/start
func Start(c *gin.Context) {
	cfg := loadConfig()
	st, err := states.New(10 * time.Minute)
	if err != nil {
		c.String(http.StatusInternalServerError, "state error")
		return
	}
	u, err := adminConsentURL(cfg.TenantID, cfg.ClientID, cfg.RedirectURI, st)
	if err != nil {
		c.String(http.StatusBadRequest, "bad config")
		return
	}
	c.Redirect(http.StatusFound, u)
}

// GET {prefix}/admin-consent/callback?admin_consent=True&tenant=...&state=...
func Callback(c *gin.Context) {
	if c.Query("admin_consent") != "True" || !states.VerifyAndDelete(c.Query("state")) {
		c.String(http.StatusBadRequest, "invalid consent or state")
		return
	}
	cbTenant := strings.TrimSpace(c.Query("tenant"))
	if cbTenant == "" {
		c.String(http.StatusBadRequest, "missing tenant")
		return
	}

	// 开发期：写入 local.env；生产建议写入安全存储
	if err := upsertLocalEnv("config/email/local.env", map[string]string{
		"EMAIL_STRATEGY":  "graph",
		"GRAPH_TENANT_ID": cbTenant,
	}); err != nil {
		c.String(http.StatusInternalServerError, "write env failed: %v", err)
		return
	}

	c.String(http.StatusOK, "Admin consent recorded for tenant: %s", cbTenant)
}

func upsertLocalEnv(p string, kv map[string]string) error {
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	old, _ := os.ReadFile(p)
	lines := strings.Split(string(old), "\n")
	index := map[string]int{}
	for i, ln := range lines {
		t := strings.TrimSpace(ln)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if k, _, ok := cutKV(t); ok {
			index[k] = i
		}
	}
	for k, v := range kv {
		if i, ok := index[k]; ok {
			lines[i] = fmt.Sprintf("%s=%s", k, v)
		} else {
			lines = append(lines, fmt.Sprintf("%s=%s", k, v))
		}
	}
	out := strings.TrimSpace(strings.Join(lines, "\n")) + "\n"
	return os.WriteFile(p, []byte(out), 0o644)
}

func cutKV(s string) (k, v string, ok bool) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

// 把路由挂在某个前缀（供外部调用）
func AttachTo(engine *gin.Engine, prefix string) {
	if prefix == "" {
		prefix = "/auth/ms"
	}
	g := engine.Group(prefix)
	g.GET("/admin-consent/start", Start)
	g.GET("/admin-consent/callback", Callback)
}
