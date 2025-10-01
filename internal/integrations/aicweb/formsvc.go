package aicweb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend-go/internal/integrations/aicweb/storage"
)

type FormService interface {
	Submit(userID, ip, ua string, payload json.RawMessage) error
	List(userID string, limit int) ([]storage.FormSubmission, error)
	Close() error
}

type sqliteFormService struct{ store *storage.SQLiteStore }

func NewFormServiceFromEnv() (FormService, error) {
	dsn := os.Getenv("AICWEB_SQLITE_PATH")
	if dsn == "" {
		// ★ 默认 databases/
		dsn = "databases/aicweb/forms.db"
	}
	// 若是本地文件路径则确保目录存在（跳过诸如 "file:" 内存DSN）
	if !strings.HasPrefix(dsn, "file:") {
		_ = os.MkdirAll(filepath.Dir(dsn), 0o755)
	}
	st, err := storage.Open(dsn)
	if err != nil {
		return nil, err
	}
	return &sqliteFormService{store: st}, nil
}

func (s *sqliteFormService) Submit(userID, ip, ua string, payload json.RawMessage) error {
	return s.store.Insert(&storage.FormSubmission{
		UserID:     userID,
		PayloadRaw: payload,
		IP:         ip,
		UserAgent:  ua,
		CreatedAt:  time.Now(),
	})
}

func (s *sqliteFormService) List(userID string, limit int) ([]storage.FormSubmission, error) {
	return s.store.ListByUser(userID, limit)
}

func (s *sqliteFormService) Close() error { return s.store.Close() }
