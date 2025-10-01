package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	_ "modernc.org/sqlite" // 纯Go驱动，无需CGO
)

type SQLiteStore struct {
	DB *sql.DB
}

func Open(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite 单文件数据库，并发上限设小一点更安全
	db.SetMaxOpenConns(1)
	s := &SQLiteStore{DB: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) Close() error { return s.DB.Close() }

func (s *SQLiteStore) migrate() error {
	const ddl = `
CREATE TABLE IF NOT EXISTS form_submissions (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id       TEXT    NOT NULL,
  payload_json  TEXT    NOT NULL,
  ip            TEXT,
  user_agent    TEXT,
  created_at    DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_form_user_created ON form_submissions(user_id, created_at DESC);
`
	_, err := s.DB.Exec(ddl)
	return err
}

type FormSubmission struct {
	ID         int64
	UserID     string
	PayloadRaw json.RawMessage
	IP         string
	UserAgent  string
	CreatedAt  time.Time
}

func (s *SQLiteStore) Insert(fs *FormSubmission) error {
	if fs == nil || len(fs.PayloadRaw) == 0 || fs.UserID == "" {
		return errors.New("invalid form submission")
	}
	stmt, err := s.DB.Prepare(`INSERT INTO form_submissions(user_id, payload_json, ip, user_agent, created_at)
		VALUES(?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	res, err := stmt.Exec(fs.UserID, string(fs.PayloadRaw), fs.IP, fs.UserAgent, fs.CreatedAt.UTC())
	if err != nil {
		return err
	}
	fs.ID, _ = res.LastInsertId()
	return nil
}

func (s *SQLiteStore) ListByUser(userID string, limit int) ([]FormSubmission, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.DB.Query(`SELECT id, payload_json, ip, user_agent, created_at
		FROM form_submissions WHERE user_id=? ORDER BY created_at DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FormSubmission
	for rows.Next() {
		var fs FormSubmission
		var payload string
		if err := rows.Scan(&fs.ID, &payload, &fs.IP, &fs.UserAgent, &fs.CreatedAt); err != nil {
			return nil, err
		}
		fs.UserID = userID
		fs.PayloadRaw = json.RawMessage(payload)
		out = append(out, fs)
	}
	return out, rows.Err()
}
