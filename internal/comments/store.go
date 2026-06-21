package comments

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("comments: not found")

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
CREATE TABLE IF NOT EXISTS comments (
  id         TEXT PRIMARY KEY,
  post_slug  TEXT NOT NULL,
  author     TEXT NOT NULL,
  content    TEXT NOT NULL,
  reply_to   TEXT NOT NULL DEFAULT '',
  ip_hash    TEXT NOT NULL DEFAULT '',
  status     TEXT NOT NULL DEFAULT 'approved',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_comments_post ON comments(post_slug, status, created_at);
CREATE INDEX IF NOT EXISTS idx_comments_reply ON comments(reply_to);
`
	_, err := s.db.Exec(ddl)
	return err
}

func (s *Store) InsertComment(ctx context.Context, c *Comment) error {
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	if c.Status == "" {
		c.Status = StatusApproved
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO comments(id, post_slug, author, content, reply_to, ip_hash, status, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?)`,
		c.ID, c.PostSlug, c.Author, c.Content, c.ReplyTo, c.IPHash, c.Status, c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *Store) ListComments(ctx context.Context, postSlug, status string, limit, offset int) ([]Comment, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var conds []string
	var args []any
	if postSlug != "" {
		conds = append(conds, "post_slug=?")
		args = append(args, postSlug)
	}
	if status != "" {
		conds = append(conds, "status=?")
		args = append(args, status)
	} else {
		conds = append(conds, "status=?")
		args = append(args, StatusApproved)
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, post_slug, author, content, reply_to, ip_hash, status, created_at, updated_at
FROM comments `+where+` ORDER BY created_at ASC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.PostSlug, &c.Author, &c.Content, &c.ReplyTo, &c.IPHash,
			&c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT count(1) FROM comments `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (s *Store) GetComment(ctx context.Context, id string) (*Comment, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, post_slug, author, content, reply_to, ip_hash, status, created_at, updated_at
FROM comments WHERE id=?`, id)
	var c Comment
	err := row.Scan(&c.ID, &c.PostSlug, &c.Author, &c.Content, &c.ReplyTo, &c.IPHash,
		&c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (s *Store) UpdateStatus(ctx context.Context, id, status string) error {
	r, err := s.db.ExecContext(ctx,
		`UPDATE comments SET status=?, updated_at=? WHERE id=?`,
		status, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteComment(ctx context.Context, id string) error {
	return s.UpdateStatus(ctx, id, StatusDeleted)
}
