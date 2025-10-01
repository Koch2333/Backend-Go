package redirect

import (
	"os"
	"path/filepath"
	"strings"

	"backend-go/internal/redirect/storage"
)

type Service struct {
	Store *storage.SQLite
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
	return &Service{Store: st}, nil
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
