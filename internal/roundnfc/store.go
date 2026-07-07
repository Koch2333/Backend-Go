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
CREATE TABLE IF NOT EXISTS nfc_writes (
  id               TEXT PRIMARY KEY,
  badge_id         TEXT NOT NULL,
  tag_uid          TEXT NOT NULL DEFAULT '',
  ndef_url         TEXT NOT NULL DEFAULT '',
  device_id        TEXT NOT NULL DEFAULT '',
  write_status     TEXT NOT NULL DEFAULT '',
  photo_object_key TEXT NOT NULL DEFAULT '',
  written_at       DATETIME NOT NULL,
  created_at       DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_nfc_writes_badge_written ON nfc_writes(badge_id, written_at);
CREATE TABLE IF NOT EXISTS app_tokens (
  id           TEXT PRIMARY KEY,
  name         TEXT NOT NULL,
  token_hash   TEXT NOT NULL UNIQUE,
  token_prefix TEXT NOT NULL DEFAULT '',
  enabled      INTEGER NOT NULL DEFAULT 1,
  last_used_at DATETIME,
  created_at   DATETIME NOT NULL,
  updated_at   DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_app_tokens_hash ON app_tokens(token_hash);
CREATE TABLE IF NOT EXISTS badge_style_templates (
  key         TEXT PRIMARY KEY,
  label       TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  image_url   TEXT NOT NULL DEFAULT '',
  payload     TEXT NOT NULL DEFAULT '{}',
  enabled     INTEGER NOT NULL DEFAULT 1,
  created_at  DATETIME NOT NULL,
  updated_at  DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS badge_coser_bindings (
  badge_id         TEXT PRIMARY KEY,
  cn               TEXT NOT NULL DEFAULT '',
  photo_object_key TEXT NOT NULL DEFAULT '',
  device_id        TEXT NOT NULL DEFAULT '',
  tag_uid          TEXT NOT NULL DEFAULT '',
  written_at       DATETIME,
  created_at       DATETIME NOT NULL,
  updated_at       DATETIME NOT NULL
);
`
	if _, err := s.db.Exec(ddl); err != nil {
		return err
	}
	if err := s.ensureColumn("badge_style_templates", "image_url", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.seedDefaultStyleTemplates(); err != nil {
		return err
	}
	return s.migrateAuth()
}

func (s *Store) ensureColumn(table, column, spec string) error {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + spec)
	return err
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
	if binding, err := s.GetBadgeCoserBinding(ctx, id); err == nil {
		b.CoserBinding = binding
	} else if err != nil && !errors.Is(err, ErrNotFound) {
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
		`SELECT b.id,b.title,b.series,b.type,b.style_key,b.image_url,b.description,b.serial_no,b.released_at,
        b.created_at,b.updated_at,
        cb.cn,cb.photo_object_key,cb.device_id,cb.tag_uid,cb.written_at,cb.created_at,cb.updated_at
FROM badges b
LEFT JOIN badge_coser_bindings cb ON cb.badge_id=b.id `+where+` ORDER BY b.updated_at DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Badge
	for rows.Next() {
		var b Badge
		var binding BadgeCoserBinding
		var cn, photoKey, deviceID, tagUID sql.NullString
		var bindingWrittenAt, bindingCreatedAt, bindingUpdatedAt sql.NullTime
		if err := rows.Scan(&b.ID, &b.Title, &b.Series, &b.Type, &b.StyleKey, &b.ImageURL,
			&b.Description, &b.SerialNo, &b.ReleasedAt, &b.CreatedAt, &b.UpdatedAt,
			&cn, &photoKey, &deviceID, &tagUID, &bindingWrittenAt, &bindingCreatedAt, &bindingUpdatedAt); err != nil {
			return nil, 0, err
		}
		if cn.Valid || photoKey.Valid {
			binding.BadgeID = b.ID
			binding.CN = cn.String
			binding.PhotoObjectKey = photoKey.String
			binding.DeviceID = deviceID.String
			binding.TagUID = tagUID.String
			if bindingWrittenAt.Valid {
				binding.WrittenAt = bindingWrittenAt.Time
			}
			if bindingCreatedAt.Valid {
				binding.CreatedAt = bindingCreatedAt.Time
			}
			if bindingUpdatedAt.Valid {
				binding.UpdatedAt = bindingUpdatedAt.Time
			}
			b.CoserBinding = &binding
		}
		out = append(out, b)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT count(1) FROM badges b `+where, args...).Scan(&total); err != nil {
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

// ----- Badge Style Templates -----

func (s *Store) seedDefaultStyleTemplates() error {
	now := time.Now().UTC()
	for _, t := range defaultBadgeStyleTemplates {
		enabled := 0
		if t.Enabled {
			enabled = 1
		}
		payload := normalizeJSON(t.Payload)
		_, err := s.db.Exec(`
INSERT INTO badge_style_templates(key,label,description,image_url,payload,enabled,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?)
ON CONFLICT(key) DO NOTHING`,
			t.Key, t.Label, t.Description, t.ImageURL, payload, enabled, now, now)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListBadgeStyleTemplates(ctx context.Context, enabledOnly bool) ([]BadgeStyleTemplate, error) {
	query := `SELECT key,label,description,image_url,payload,enabled,created_at,updated_at FROM badge_style_templates`
	if enabledOnly {
		query += ` WHERE enabled=1`
	}
	query += ` ORDER BY key`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BadgeStyleTemplate
	for rows.Next() {
		var t BadgeStyleTemplate
		var payload string
		var enabled int
		if err := rows.Scan(&t.Key, &t.Label, &t.Description, &t.ImageURL, &payload, &enabled, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Payload = json.RawMessage(payload)
		t.Enabled = enabled == 1
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetBadgeStyleTemplate(ctx context.Context, key string) (*BadgeStyleTemplate, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT key,label,description,image_url,payload,enabled,created_at,updated_at
FROM badge_style_templates WHERE key=?`, key)
	var t BadgeStyleTemplate
	var payload string
	var enabled int
	if err := row.Scan(&t.Key, &t.Label, &t.Description, &t.ImageURL, &payload, &enabled, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t.Payload = json.RawMessage(payload)
	t.Enabled = enabled == 1
	return &t, nil
}

func (s *Store) UpsertBadgeStyleTemplate(ctx context.Context, t *BadgeStyleTemplate) error {
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	enabled := 0
	if t.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO badge_style_templates(key,label,description,image_url,payload,enabled,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?)
ON CONFLICT(key) DO UPDATE SET
  label=excluded.label,
  description=excluded.description,
  image_url=excluded.image_url,
  payload=excluded.payload,
  enabled=excluded.enabled,
  updated_at=excluded.updated_at`,
		t.Key, t.Label, t.Description, t.ImageURL, normalizeJSON(t.Payload), enabled, t.CreatedAt, t.UpdatedAt)
	return err
}

func (s *Store) DeleteBadgeStyleTemplate(ctx context.Context, key string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM badge_style_templates WHERE key=?`, key)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ValidBadgeStyleKey(ctx context.Context, key string) (bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return true, nil
	}
	var enabled int
	err := s.db.QueryRowContext(ctx, `SELECT enabled FROM badge_style_templates WHERE key=?`, key).Scan(&enabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return enabled == 1, nil
}

func normalizeJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// ----- Coser Bindings -----

func (s *Store) UpsertBadgeCoserBinding(ctx context.Context, b *BadgeCoserBinding) error {
	now := time.Now().UTC()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	b.UpdatedAt = now
	_, err := s.db.ExecContext(ctx, `
INSERT INTO badge_coser_bindings(badge_id,cn,photo_object_key,device_id,tag_uid,written_at,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?)
ON CONFLICT(badge_id) DO UPDATE SET
  cn=excluded.cn,
  photo_object_key=excluded.photo_object_key,
  device_id=excluded.device_id,
  tag_uid=excluded.tag_uid,
  written_at=excluded.written_at,
  updated_at=excluded.updated_at`,
		b.BadgeID, b.CN, b.PhotoObjectKey, b.DeviceID, b.TagUID, nullableTime(b.WrittenAt), b.CreatedAt, b.UpdatedAt)
	return err
}

func (s *Store) GetBadgeCoserBinding(ctx context.Context, badgeID string) (*BadgeCoserBinding, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT badge_id,cn,photo_object_key,device_id,tag_uid,written_at,created_at,updated_at
FROM badge_coser_bindings WHERE badge_id=?`, badgeID)
	var b BadgeCoserBinding
	var written sql.NullTime
	if err := row.Scan(&b.BadgeID, &b.CN, &b.PhotoObjectKey, &b.DeviceID, &b.TagUID, &written, &b.CreatedAt, &b.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if written.Valid {
		b.WrittenAt = written.Time
	}
	return &b, nil
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// ----- App Tokens -----

func (s *Store) ListAppTokens(ctx context.Context) ([]AppToken, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id,name,token_prefix,enabled,last_used_at,created_at,updated_at
FROM app_tokens ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AppToken
	for rows.Next() {
		var t AppToken
		var enabled int
		var last sql.NullTime
		if err := rows.Scan(&t.ID, &t.Name, &t.TokenPrefix, &enabled, &last, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Enabled = enabled == 1
		if last.Valid {
			t.LastUsedAt = &last.Time
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) InsertAppToken(ctx context.Context, t *AppToken, tokenHash string) error {
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	enabled := 0
	if t.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO app_tokens(id,name,token_hash,token_prefix,enabled,created_at,updated_at)
VALUES(?,?,?,?,?,?,?)`,
		t.ID, t.Name, tokenHash, t.TokenPrefix, enabled, t.CreatedAt, t.UpdatedAt)
	return err
}

func (s *Store) SetAppTokenEnabled(ctx context.Context, id string, enabled bool) error {
	flag := 0
	if enabled {
		flag = 1
	}
	res, err := s.db.ExecContext(ctx, `UPDATE app_tokens SET enabled=?, updated_at=? WHERE id=?`,
		flag, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteAppToken(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM app_tokens WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) VerifyAppToken(ctx context.Context, tokenHash string) (bool, error) {
	var id string
	var enabled int
	err := s.db.QueryRowContext(ctx, `SELECT id,enabled FROM app_tokens WHERE token_hash=?`, tokenHash).
		Scan(&id, &enabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if enabled != 1 {
		return false, nil
	}
	now := time.Now().UTC()
	_, _ = s.db.ExecContext(ctx, `UPDATE app_tokens SET last_used_at=?, updated_at=? WHERE id=?`, now, now, id)
	return true, nil
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

// ----- NFC Writes -----

func (s *Store) InsertNFCWrite(ctx context.Context, w *NFCWrite) error {
	now := time.Now().UTC()
	if w.ID == "" {
		w.ID = newID("nfcw")
	}
	if w.WrittenAt.IsZero() {
		w.WrittenAt = now
	}
	w.CreatedAt = now
	_, err := s.db.ExecContext(ctx, `
INSERT INTO nfc_writes(id,badge_id,tag_uid,ndef_url,device_id,write_status,photo_object_key,written_at,created_at)
VALUES(?,?,?,?,?,?,?,?,?)`,
		w.ID, w.BadgeID, w.TagUID, w.NDEFURL, w.DeviceID, w.WriteStatus, w.PhotoObjectKey, w.WrittenAt, w.CreatedAt)
	return err
}
