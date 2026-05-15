package storage

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type SQLite struct{ DB *sql.DB }

func Open(dsn string) (*SQLite, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &SQLite{DB: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.MigrateAuth(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// ----- listing & deletion (admin) -----

type Rule struct {
	Name      string
	TargetURL string
	Enabled   bool
	UpdatedAt time.Time
}

func (s *SQLite) ListRules(q string, limit, offset int) ([]Rule, int, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	args := []any{}
	where := ""
	if q != "" {
		where = " WHERE name LIKE ? OR target_url LIKE ?"
		args = append(args, "%"+q+"%", "%"+q+"%")
	}
	var total int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM redirect_rules`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.DB.Query(`SELECT name, target_url, enabled, updated_at FROM redirect_rules`+where+` ORDER BY name LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Rule
	for rows.Next() {
		var r Rule
		var e int
		if err := rows.Scan(&r.Name, &r.TargetURL, &e, &r.UpdatedAt); err != nil {
			return nil, 0, err
		}
		r.Enabled = e == 1
		out = append(out, r)
	}
	return out, total, nil
}

func (s *SQLite) DeleteRule(name string) error {
	_, err := s.DB.Exec(`DELETE FROM redirect_rules WHERE name=?`, name)
	return err
}

func (s *SQLite) ListCards(q string, limit, offset int) ([]NFCCard, int, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	args := []any{}
	where := ""
	if q != "" {
		where = " WHERE hwid LIKE ? OR user_id LIKE ?"
		args = append(args, "%"+q+"%", "%"+q+"%")
	}
	var total int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM nfc_cards`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.DB.Query(`SELECT hwid, is_registered, user_id, updated_at FROM nfc_cards`+where+` ORDER BY updated_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []NFCCard
	for rows.Next() {
		var c NFCCard
		var reg int
		if err := rows.Scan(&c.HWID, &reg, &c.UserID, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		c.IsRegistered = reg == 1
		out = append(out, c)
	}
	return out, total, nil
}

func (s *SQLite) DeleteCard(hwid string) error {
	_, err := s.DB.Exec(`DELETE FROM nfc_cards WHERE hwid=?`, hwid)
	return err
}

func (s *SQLite) Close() error { return s.DB.Close() }

func (s *SQLite) migrate() error {
	const ddl = `
CREATE TABLE IF NOT EXISTS redirect_rules (
  name        TEXT PRIMARY KEY,
  target_url  TEXT NOT NULL,
  enabled     INTEGER NOT NULL DEFAULT 1,
  updated_at  DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS nfc_cards (
  hwid         TEXT PRIMARY KEY,
  is_registered INTEGER NOT NULL DEFAULT 0, -- 0=false,1=true
  user_id      TEXT,
  updated_at   DATETIME NOT NULL
);
`
	_, err := s.DB.Exec(ddl)
	return err
}

// ----- redirect_rules -----

func (s *SQLite) ResolveRule(name string) (target string, enabled bool, found bool, err error) {
	row := s.DB.QueryRow(`SELECT target_url, enabled FROM redirect_rules WHERE name=?`, name)
	var e int
	if err = row.Scan(&target, &e); err != nil {
		if err == sql.ErrNoRows {
			return "", false, false, nil
		}
		return "", false, false, err
	}
	return target, e == 1, true, nil
}

func (s *SQLite) UpsertRule(name, url string, enabled bool) error {
	_, err := s.DB.Exec(`
INSERT INTO redirect_rules(name, target_url, enabled, updated_at)
VALUES(?,?,?,?)
ON CONFLICT(name) DO UPDATE SET target_url=excluded.target_url, enabled=excluded.enabled, updated_at=excluded.updated_at
`, name, url, boolToInt(enabled), time.Now().UTC())
	return err
}

// ----- nfc_cards -----

type NFCCard struct {
	HWID         string
	IsRegistered bool
	UserID       string
	UpdatedAt    time.Time
}

func (s *SQLite) GetCard(hwid string) (*NFCCard, error) {
	row := s.DB.QueryRow(`SELECT hwid, is_registered, user_id, updated_at FROM nfc_cards WHERE hwid=?`, hwid)
	var c NFCCard
	var reg int
	if err := row.Scan(&c.HWID, &reg, &c.UserID, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	c.IsRegistered = reg == 1
	return &c, nil
}

func (s *SQLite) UpsertCard(hwid string, isRegistered bool, userID string) error {
	_, err := s.DB.Exec(`
INSERT INTO nfc_cards(hwid, is_registered, user_id, updated_at)
VALUES(?,?,?,?)
ON CONFLICT(hwid) DO UPDATE SET is_registered=excluded.is_registered, user_id=excluded.user_id, updated_at=excluded.updated_at
`, hwid, boolToInt(isRegistered), userID, time.Now().UTC())
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
