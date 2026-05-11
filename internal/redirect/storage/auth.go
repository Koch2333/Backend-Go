package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"backend-go/internal/authflow"
)

func (s *SQLite) MigrateAuth() error {
	const ddl = `
CREATE TABLE IF NOT EXISTS admin_totp (
    username   TEXT PRIMARY KEY,
    secret     TEXT NOT NULL DEFAULT '',
    enabled    INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS admin_passkeys (
    id         TEXT PRIMARY KEY,
    username   TEXT NOT NULL,
    name       TEXT NOT NULL DEFAULT '',
    public_key BLOB NOT NULL,
    counter    INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_passkeys_username ON admin_passkeys(username);
`
	_, err := s.DB.Exec(ddl)
	return err
}

func (s *SQLite) GetTOTP(username string) (string, bool, error) {
	var secret string
	var enabled int
	err := s.DB.QueryRowContext(context.Background(),
		`SELECT secret, enabled FROM admin_totp WHERE username=?`, username).
		Scan(&secret, &enabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return secret, enabled == 1, nil
}

func (s *SQLite) SetTOTP(username, secret string, enabled bool) error {
	e := 0
	if enabled {
		e = 1
	}
	_, err := s.DB.ExecContext(context.Background(), `
INSERT INTO admin_totp(username, secret, enabled, updated_at) VALUES(?, ?, ?, ?)
ON CONFLICT(username) DO UPDATE SET
  secret=excluded.secret, enabled=excluded.enabled, updated_at=excluded.updated_at`,
		username, secret, e, time.Now().UTC())
	return err
}

func (s *SQLite) GetCredentials(username string) ([]authflow.Credential, error) {
	rows, err := s.DB.QueryContext(context.Background(),
		`SELECT id, username, name, public_key, counter, created_at
		 FROM admin_passkeys WHERE username=? ORDER BY created_at`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []authflow.Credential
	for rows.Next() {
		var c authflow.Credential
		if err := rows.Scan(&c.ID, &c.Username, &c.Name, &c.PublicKey, &c.Counter, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *SQLite) SaveCredential(c *authflow.Credential) error {
	_, err := s.DB.ExecContext(context.Background(),
		`INSERT INTO admin_passkeys(id, username, name, public_key, counter, created_at)
		 VALUES(?, ?, ?, ?, ?, ?)`,
		c.ID, c.Username, c.Name, c.PublicKey, c.Counter, c.CreatedAt)
	return err
}

func (s *SQLite) DeleteCredential(username, credID string) error {
	_, err := s.DB.ExecContext(context.Background(),
		`DELETE FROM admin_passkeys WHERE username=? AND id=?`, username, credID)
	return err
}

func (s *SQLite) UpdateCounter(credID string, counter uint32) error {
	_, err := s.DB.ExecContext(context.Background(),
		`UPDATE admin_passkeys SET counter=? WHERE id=?`, counter, credID)
	return err
}

func (s *SQLite) ListCredentials(username string) ([]authflow.CredentialInfo, error) {
	rows, err := s.DB.QueryContext(context.Background(),
		`SELECT id, name, created_at FROM admin_passkeys WHERE username=? ORDER BY created_at`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []authflow.CredentialInfo
	for rows.Next() {
		var c authflow.CredentialInfo
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}
