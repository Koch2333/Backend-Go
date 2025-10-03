package redirect

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

// Store 仅负责数据库读写，不做路由或注册
type Store struct {
	db *sql.DB
}

// Redirect 普通重定向记录：name -> url
type Redirect struct {
	Name      string
	URL       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PNCS 记录：NFC/PNCS 卡关联信息
type PNCS struct {
	HWID         string
	IsRegistered bool
	UserID       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewStoreFromEnv 初始化存储；REDIRECT_SQLITE_PATH 为空则使用默认路径
func NewStoreFromEnv() (*Store, error) {
	dsn := strings.TrimSpace(os.Getenv("REDIRECT_SQLITE_PATH"))
	if dsn == "" {
		dsn = "databases/redirect/redirect.db"
	}
	// 确保目录存在（兼容 file:xxx?cache=shared 这类 DSN）
	_ = os.MkdirAll(filepath.Dir(extractSQLiteFilePath(dsn)), 0o755)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// ---------- schema ----------

func migrate(db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS redirects (
  name        TEXT PRIMARY KEY,
  url         TEXT NOT NULL,
  created_at  DATETIME NOT NULL,
  updated_at  DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS pncs (
  hwid          TEXT PRIMARY KEY,
  is_registered INTEGER NOT NULL DEFAULT 0,
  user_id       TEXT,
  created_at    DATETIME NOT NULL,
  updated_at    DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_redirects_updated ON redirects(updated_at);
CREATE INDEX IF NOT EXISTS idx_pncs_updated ON pncs(updated_at);
`
	_, err := db.Exec(ddl)
	return err
}

// ---------- Redirects: name <-> url ----------

// UpsertRedirect 新增或覆盖普通重定向
func (s *Store) UpsertRedirect(ctx context.Context, name, url string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO redirects(name, url, created_at, updated_at)
VALUES(?,?,?,?)
ON CONFLICT(name) DO UPDATE SET
  url=excluded.url,
  updated_at=excluded.updated_at
`, name, url, now, now)
	return err
}

// GetURLByName 通过名称查 URL
func (s *Store) GetURLByName(ctx context.Context, name string) (string, error) {
	var url string
	err := s.db.QueryRowContext(ctx, `SELECT url FROM redirects WHERE name=?`, name).Scan(&url)
	if errors.Is(err, sql.ErrNoRows) {
		return "", sql.ErrNoRows
	}
	return url, err
}

// DeleteRedirect 删除一条重定向
func (s *Store) DeleteRedirect(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM redirects WHERE name=?`, name)
	return err
}

// ---------- PNCS: hwid / is_registered / user_id ----------

// UpsertPNCS 新增或覆盖一条 PNCS 记录
func (s *Store) UpsertPNCS(ctx context.Context, hwid string, isRegistered bool, userID string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO pncs(hwid, is_registered, user_id, created_at, updated_at)
VALUES(?,?,?,?,?)
ON CONFLICT(hwid) DO UPDATE SET
  is_registered=excluded.is_registered,
  user_id=excluded.user_id,
  updated_at=excluded.updated_at
`, hwid, b2i(isRegistered), userID, now, now)
	return err
}

// GetPNCS 按 hwid 读取一条 PNCS 记录
func (s *Store) GetPNCS(ctx context.Context, hwid string) (*PNCS, error) {
	var (
		p  PNCS
		ir int
	)
	row := s.db.QueryRowContext(ctx, `
SELECT hwid, is_registered, user_id, created_at, updated_at
FROM pncs WHERE hwid=?`, hwid)
	if err := row.Scan(&p.HWID, &ir, &p.UserID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	p.IsRegistered = ir != 0
	return &p, nil
}

// SetPNCSRegistered 设置激活状态
func (s *Store) SetPNCSRegistered(ctx context.Context, hwid string, isRegistered bool) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE pncs SET is_registered=?, updated_at=? WHERE hwid=?`,
		b2i(isRegistered), time.Now().UTC(), hwid)
	return err
}

// BindPNCSUser 绑定用户 ID（例如 aicweb 注册后回填）
func (s *Store) BindPNCSUser(ctx context.Context, hwid, userID string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE pncs SET user_id=?, updated_at=? WHERE hwid=?`,
		userID, time.Now().UTC(), hwid)
	return err
}

// ---------- helpers ----------

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// 提取真正的文件路径，兼容 DSN 形态：file:xxx.db?cache=shared
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
