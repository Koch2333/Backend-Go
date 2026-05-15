package redirect

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"backend-go/internal/authflow"
	"backend-go/internal/redirect/storage"
)

type AdminConfig struct {
	Username     string
	PasswordHash string
	JWTSecret    []byte
	JWTTTL       time.Duration
	TOTPIssuer   string
	WARPID       string
	WARPName     string
	WAOrigins    []string
}

type Service struct {
	Store *storage.SQLite
	Admin AdminConfig
}

func NewServiceFromEnv() (*Service, error) {
	dsn := os.Getenv("REDIRECT_SQLITE_PATH")
	if dsn == "" {
		// ★ 默认 databases/
		dsn = "databases/redirect/redirect.db"
	}
	if !strings.HasPrefix(dsn, "file:") {
		_ = os.MkdirAll(filepath.Dir(dsn), 0o755)
	}
	st, err := storage.Open(dsn)
	if err != nil {
		return nil, err
	}
	return &Service{Store: st, Admin: loadAdminConfig()}, nil
}

func loadAdminConfig() AdminConfig {
	ttl := 12 * time.Hour
	if v := strings.TrimSpace(os.Getenv("REDIRECT_JWT_TTL_HOURS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttl = time.Duration(n) * time.Hour
		}
	}
	originsRaw := os.Getenv("REDIRECT_WEBAUTHN_ORIGINS")
	if strings.TrimSpace(originsRaw) == "" {
		originsRaw = "http://localhost:5174,http://localhost:8081"
	}
	var origins []string
	for _, o := range strings.Split(originsRaw, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	username := strings.TrimSpace(os.Getenv("REDIRECT_ADMIN_USERNAME"))
	if username == "" {
		username = "admin"
	}
	issuer := strings.TrimSpace(os.Getenv("REDIRECT_TOTP_ISSUER"))
	if issuer == "" {
		issuer = "Redirect"
	}
	rpName := strings.TrimSpace(os.Getenv("REDIRECT_WEBAUTHN_RP_NAME"))
	if rpName == "" {
		rpName = "Redirect Admin"
	}
	rpID := strings.TrimSpace(os.Getenv("REDIRECT_WEBAUTHN_RPID"))
	if rpID == "" {
		rpID = "localhost"
	}
	return AdminConfig{
		Username:     username,
		PasswordHash: strings.TrimSpace(os.Getenv("REDIRECT_ADMIN_PASSWORD_HASH")),
		JWTSecret:    []byte(strings.TrimSpace(os.Getenv("REDIRECT_JWT_SECRET"))),
		JWTTTL:       ttl,
		TOTPIssuer:   issuer,
		WARPID:       rpID,
		WARPName:     rpName,
		WAOrigins:    origins,
	}
}

// AuthFlowConfig returns the authflow.Config for this service.
func (s *Service) AuthFlowConfig() authflow.Config {
	return authflow.Config{
		Store:             s.Store,
		AdminUsername:     s.Admin.Username,
		AdminPasswordHash: s.Admin.PasswordHash,
		JWTSecret:         s.Admin.JWTSecret,
		JWTTTL:            s.Admin.JWTTTL,
		TOTPIssuer:        s.Admin.TOTPIssuer,
		WebAuthnRPID:      s.Admin.WARPID,
		WebAuthnRPName:    s.Admin.WARPName,
		WebAuthnOrigins:   s.Admin.WAOrigins,
	}
}

func (s *Service) Close() error { return s.Store.Close() }

func (s *Service) ResolveByName(name string) (string, bool, error) {
	url, enabled, found, err := s.Store.ResolveRule(name)
	if err != nil {
		return "", false, err
	}
	if found && enabled {
		return url, true, nil
	}
	return s.expand(os.Getenv("REDIRECT_NOT_FOUND_URL"), map[string]string{"name": name}), false, nil
}

func (s *Service) ResolveNFC(hwid string) (string, error) {
	card, err := s.Store.GetCard(hwid)
	if err != nil {
		return "", err
	}
	if card != nil && card.IsRegistered {
		return s.expand(os.Getenv("REDIRECT_NFC_REGISTERED_URL"), map[string]string{
			"hwid":       card.HWID,
			"userId":     card.UserID,
			"registered": "true",
		}), nil
	}
	return s.expand(os.Getenv("REDIRECT_NFC_UNREGISTERED_URL"), map[string]string{
		"hwid":       hwid,
		"registered": "false",
	}), nil
}

func (s *Service) UpsertCard(hwid string, isRegistered bool, userID string) error {
	return s.Store.UpsertCard(hwid, isRegistered, userID)
}
func (s *Service) UpsertRule(name, url string, enabled bool) error {
	return s.Store.UpsertRule(name, url, enabled)
}

func (s *Service) expand(tpl string, vars map[string]string) string {
	if tpl == "" {
		if _, ok := vars["hwid"]; ok {
			tpl = "https://koch2333.cn/pncs/register?hwid={hwid}"
		} else {
			tpl = "https://koch2333.cn/404?name={name}"
		}
	}
	out := tpl
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}
