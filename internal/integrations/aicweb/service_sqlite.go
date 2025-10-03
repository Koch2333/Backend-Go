package aicweb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type sqliteService struct {
	db     *sql.DB
	mu     sync.RWMutex
	tokens map[string]string // session token -> email
}

func NewServiceSQLiteFromEnv() (Service, error) {
	dsn := strings.TrimSpace(os.Getenv("AICWEB_USERS_SQLITE_PATH"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("AICWEB_SQLITE_PATH"))
	}
	if dsn == "" {
		dsn = "databases/aicweb/users.db"
	}
	_ = os.MkdirAll(filepath.Dir(extractSQLiteFilePath(dsn)), 0o755)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := migrateUsers(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &sqliteService{db: db, tokens: map[string]string{}}, nil
}

func migrateUsers(db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS users (
  id            TEXT PRIMARY KEY,
  email         TEXT UNIQUE NOT NULL,
  username      TEXT,
  password_hash TEXT NOT NULL,
  is_registered INTEGER NOT NULL DEFAULT 0,
  created_at    DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE TABLE IF NOT EXISTS activation_tokens (
  token       TEXT PRIMARY KEY,
  user_id     TEXT NOT NULL,
  email       TEXT NOT NULL,
  expires_at  DATETIME NOT NULL,
  used_at     DATETIME,
  created_at  DATETIME NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id)
);
`
	_, err := db.Exec(ddl)
	return err
}

func (s *sqliteService) Register(ctx context.Context, req *RegisterRequest) error {
	var exists int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE email=?`, req.Email).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return ErrEmailAlreadyUse
	}
	var next int64
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(CAST(id AS INTEGER)), 9999) + 1 FROM users`).Scan(&next); err != nil {
		return err
	}
	id := strconv.FormatInt(next, 10)
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO users(id,email,username,password_hash,is_registered,created_at)
		VALUES(?,?,?,?,0,?)`, id, req.Email, req.Username, string(hash), time.Now().UTC())
	return err
}

func (s *sqliteService) Login(ctx context.Context, req *LoginRequest) (string, error) {
	var row struct {
		id, email, username, hash string
		isReg                     int
		created                   time.Time
	}
	q := `SELECT id,email,username,password_hash,is_registered,created_at FROM users WHERE `
	var arg any
	if req.Email != "" {
		q += `email=?`
		arg = req.Email
	} else {
		q += `username=?`
		arg = req.Username
	}
	if err := s.db.QueryRow(q, arg).Scan(&row.id, &row.email, &row.username, &row.hash, &row.isReg, &row.created); err != nil {
		return "", ErrUnauthorized
	}
	// ★ 未激活：给上层一个可判别的错误
	if row.isReg == 0 {
		return "", ErrNotActivated
	}
	if bcrypt.CompareHashAndPassword([]byte(row.hash), []byte(req.Password)) != nil {
		return "", ErrUnauthorized
	}
	tok := randHex(32)
	s.mu.Lock()
	s.tokens[tok] = row.email
	s.mu.Unlock()
	return tok, nil
}

func (s *sqliteService) Validate(ctx context.Context, token string) (*user, error) {
	s.mu.RLock()
	email, ok := s.tokens[token]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrUnauthorized
	}
	var row struct {
		id, username string
		created      time.Time
		isReg        int
	}
	if err := s.db.QueryRow(`SELECT id,username,created_at,is_registered FROM users WHERE email=?`, email).
		Scan(&row.id, &row.username, &row.created, &row.isReg); err != nil || row.isReg == 0 {
		return nil, ErrUnauthorized
	}
	return &user{ID: row.id, Username: row.username, Email: email, CreatedAt: row.created}, nil
}

// 激活：生成 token / 激活用户（保持不变）
func (s *sqliteService) CreateActivationToken(ctx context.Context, email string) (string, error) {
	var uid string
	if err := s.db.QueryRow(`SELECT id FROM users WHERE email=?`, email).Scan(&uid); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrUnauthorized
		}
		return "", err
	}
	ttl := 24
	if v := strings.TrimSpace(os.Getenv("AICWEB_ACTIVATION_TTL_HOURS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 168 {
			ttl = n
		}
	}
	token := randHex(32)
	_, err := s.db.Exec(`INSERT INTO activation_tokens(token,user_id,email,expires_at,created_at)
		VALUES(?,?,?,?,?)`, token, uid, email, time.Now().UTC().Add(time.Duration(ttl)*time.Hour), time.Now().UTC())
	return token, err
}

func (s *sqliteService) ActivateByToken(ctx context.Context, token string) error {
	var uid string
	var exp, used sql.NullTime
	if err := s.db.QueryRow(`SELECT user_id,expires_at,used_at FROM activation_tokens WHERE token=?`, token).
		Scan(&uid, &exp, &used); err != nil {
		return ErrUnauthorized
	}
	if used.Valid || time.Now().UTC().After(exp.Time) {
		return ErrUnauthorized
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.Exec(`UPDATE users SET is_registered=1 WHERE id=?`, uid); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE activation_tokens SET used_at=? WHERE token=?`, time.Now().UTC(), token); err != nil {
		return err
	}
	return tx.Commit()
}

func randHex(n int) string { b := make([]byte, n); _, _ = rand.Read(b); return hex.EncodeToString(b) }

func extractSQLiteFilePath(dsn string) string {
	if strings.HasPrefix(dsn, "file:") {
		rest := strings.TrimPrefix(dsn, "file:")
		if i := strings.IndexByte(rest, '?'); i >= 0 {
			rest = rest[:i]
		}
		rest = strings.TrimPrefix(rest, "///")
		return rest
	}
	if i := strings.IndexByte(dsn, '?'); i >= 0 {
		dsn = dsn[:i]
	}
	return dsn
}
