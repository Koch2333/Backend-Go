package roundnfc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("roundnfc: not found")

type Store struct{ db *sql.DB }

func openStore(dsn string) (*Store, error) {
	if !strings.HasPrefix(dsn, "file:") {
		_ = os.MkdirAll(filepath.Dir(dsn), 0o755)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate() error {
	const ddl = `
CREATE TABLE IF NOT EXISTS badges (
  id          TEXT PRIMARY KEY,
  title       TEXT NOT NULL DEFAULT '',
  series      TEXT NOT NULL DEFAULT '',
  type        TEXT NOT NULL DEFAULT '',
  style_key   TEXT NOT NULL DEFAULT '',
  image_url   TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  serial_no   TEXT NOT NULL DEFAULT '',
  released_at TEXT NOT NULL DEFAULT '',
  created_at  DATETIME NOT NULL,
  updated_at  DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS photo_requests (
  id              TEXT PRIMARY KEY,
  badge_id        TEXT NOT NULL,
  name            TEXT NOT NULL,
  contact         TEXT NOT NULL,
  message         TEXT NOT NULL DEFAULT '',
  status          TEXT NOT NULL DEFAULT 'new',
  attachment_keys TEXT NOT NULL DEFAULT '[]',
  ip_hash         TEXT NOT NULL DEFAULT '',
  created_at      DATETIME NOT NULL,
  updated_at      DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_photo_badge_status ON photo_requests(badge_id, status, updated_at);
CREATE TABLE IF NOT EXISTS autograph_requests (
  id              TEXT PRIMARY KEY,
  badge_id        TEXT NOT NULL,
  name            TEXT NOT NULL,
  contact         TEXT NOT NULL,
  target          TEXT NOT NULL DEFAULT '',
  content         TEXT NOT NULL DEFAULT '',
  status          TEXT NOT NULL DEFAULT 'new',
  attachment_keys TEXT NOT NULL DEFAULT '[]',
  ip_hash         TEXT NOT NULL DEFAULT '',
  created_at      DATETIME NOT NULL,
  updated_at      DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_auto_badge_status ON autograph_requests(badge_id, status, updated_at);
`
	if _, err := s.db.Exec(ddl); err != nil {
		return err
	}
	return s.migrateAuth()
}

// ----- Badges -----

func (s *Store) GetBadge(ctx context.Context, id string) (*Badge, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id,title,series,type,style_key,image_url,description,serial_no,released_at,created_at,updated_at
FROM badges WHERE id=?`, id)
	var b Badge
	err := row.Scan(&b.ID, &b.Title, &b.Series, &b.Type, &b.StyleKey, &b.ImageURL,
		&b.Description, &b.SerialNo, &b.ReleasedAt, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (s *Store) ListBadges(ctx context.Context, q string, limit, offset int) ([]Badge, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var (
		where string
		args  []any
	)
	if q != "" {
		where = `WHERE id LIKE ? OR title LIKE ? OR series LIKE ?`
		wild := "%" + q + "%"
		args = append(args, wild, wild, wild)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id,title,series,type,style_key,image_url,description,serial_no,released_at,created_at,updated_at
FROM badges `+where+` ORDER BY updated_at DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Badge
	for rows.Next() {
		var b Badge
		if err := rows.Scan(&b.ID, &b.Title, &b.Series, &b.Type, &b.StyleKey, &b.ImageURL,
			&b.Description, &b.SerialNo, &b.ReleasedAt, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, b)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT count(1) FROM badges `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (s *Store) UpsertBadge(ctx context.Context, b *Badge) error {
	now := time.Now().UTC()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	b.UpdatedAt = now
	_, err := s.db.ExecContext(ctx, `
INSERT INTO badges(id,title,series,type,style_key,image_url,description,serial_no,released_at,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  title=excluded.title,
  series=excluded.series,
  type=excluded.type,
  style_key=excluded.style_key,
  image_url=excluded.image_url,
  description=excluded.description,
  serial_no=excluded.serial_no,
  released_at=excluded.released_at,
  updated_at=excluded.updated_at`,
		b.ID, b.Title, b.Series, b.Type, b.StyleKey, b.ImageURL, b.Description,
		b.SerialNo, b.ReleasedAt, b.CreatedAt, b.UpdatedAt)
	return err
}

func (s *Store) DeleteBadge(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM badges WHERE id=?`, id)
	return err
}

// ----- helpers -----

func encodeKeys(keys []string) string {
	if len(keys) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(keys)
	return string(b)
}

func decodeKeys(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

// ----- Photo Requests -----

func (s *Store) InsertPhotoRequest(ctx context.Context, p *PhotoRequest) error {
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = StatusNew
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO photo_requests(id,badge_id,name,contact,message,status,attachment_keys,ip_hash,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.BadgeID, p.Name, p.Contact, p.Message, p.Status, encodeKeys(p.AttachmentKeys),
		p.IPHash, p.CreatedAt, p.UpdatedAt)
	return err
}

func (s *Store) ListPhotoRequests(ctx context.Context, badgeID, status string, limit, offset int) ([]PhotoRequest, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var conds []string
	var args []any
	if badgeID != "" {
		conds = append(conds, "badge_id=?")
		args = append(args, badgeID)
	}
	if status != "" {
		conds = append(conds, "status=?")
		args = append(args, status)
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id,badge_id,name,contact,message,status,attachment_keys,ip_hash,created_at,updated_at
FROM photo_requests `+where+` ORDER BY updated_at DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []PhotoRequest
	for rows.Next() {
		var p PhotoRequest
		var keys string
		if err := rows.Scan(&p.ID, &p.BadgeID, &p.Name, &p.Contact, &p.Message, &p.Status,
			&keys, &p.IPHash, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		p.AttachmentKeys = decodeKeys(keys)
		out = append(out, p)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT count(1) FROM photo_requests `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (s *Store) UpdatePhotoStatus(ctx context.Context, id, status string) error {
	r, err := s.db.ExecContext(ctx,
		`UPDATE photo_requests SET status=?, updated_at=? WHERE id=?`,
		status, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ----- Autograph Requests -----

func (s *Store) InsertAutographRequest(ctx context.Context, p *AutographRequest) error {
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = StatusNew
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO autograph_requests(id,badge_id,name,contact,target,content,status,attachment_keys,ip_hash,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.BadgeID, p.Name, p.Contact, p.Target, p.Content, p.Status,
		encodeKeys(p.AttachmentKeys), p.IPHash, p.CreatedAt, p.UpdatedAt)
	return err
}

func (s *Store) ListAutographRequests(ctx context.Context, badgeID, status string, limit, offset int) ([]AutographRequest, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var conds []string
	var args []any
	if badgeID != "" {
		conds = append(conds, "badge_id=?")
		args = append(args, badgeID)
	}
	if status != "" {
		conds = append(conds, "status=?")
		args = append(args, status)
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id,badge_id,name,contact,target,content,status,attachment_keys,ip_hash,created_at,updated_at
FROM autograph_requests `+where+` ORDER BY updated_at DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []AutographRequest
	for rows.Next() {
		var p AutographRequest
		var keys string
		if err := rows.Scan(&p.ID, &p.BadgeID, &p.Name, &p.Contact, &p.Target, &p.Content,
			&p.Status, &keys, &p.IPHash, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		p.AttachmentKeys = decodeKeys(keys)
		out = append(out, p)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT count(1) FROM autograph_requests `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (s *Store) UpdateAutographStatus(ctx context.Context, id, status string) error {
	r, err := s.db.ExecContext(ctx,
		`UPDATE autograph_requests SET status=?, updated_at=? WHERE id=?`,
		status, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
