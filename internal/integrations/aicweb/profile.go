package aicweb

import (
	"context"
	"database/sql"
	"time"
)

// PublicProfile is the publicly visible profile of a user.
type PublicProfile struct {
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	Bio         string    `json:"bio"`
	GitHubName  string    `json:"githubName"`
	BilibiliUID string    `json:"bilibiliUid"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ProfileUpdate is the writable portion of a user's profile.
type ProfileUpdate struct {
	DisplayName string `json:"displayName"`
	Bio         string `json:"bio"`
	GitHubName  string `json:"githubName"`
	BilibiliUID string `json:"bilibiliUid"`
}

// ProfileService is optionally implemented by Service backends that
// store persistent profile data (the memory backend does not).
type ProfileService interface {
	ListPublicProfiles(ctx context.Context) ([]PublicProfile, error)
	GetPublicProfile(ctx context.Context, username string) (*PublicProfile, error)
	UpdateMyProfile(ctx context.Context, userID string, update ProfileUpdate) error
}

// Compile-time check: sqliteService must implement ProfileService.
var _ ProfileService = (*sqliteService)(nil)

const profileDDL = `
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id      TEXT PRIMARY KEY,
    display_name TEXT NOT NULL DEFAULT '',
    bio          TEXT NOT NULL DEFAULT '',
    github_name  TEXT NOT NULL DEFAULT '',
    bilibili_uid TEXT NOT NULL DEFAULT '',
    updated_at   DATETIME NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_user_profiles_uid ON user_profiles(user_id);
`

func migrateProfiles(db *sql.DB) error {
	_, err := db.Exec(profileDDL)
	return err
}

func (s *sqliteService) ListPublicProfiles(ctx context.Context) ([]PublicProfile, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.username,
		       COALESCE(p.display_name, ''),
		       COALESCE(p.bio, ''),
		       COALESCE(p.github_name, ''),
		       COALESCE(p.bilibili_uid, ''),
		       COALESCE(p.updated_at, u.created_at)
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.is_registered = 1
		ORDER BY u.username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PublicProfile
	for rows.Next() {
		var p PublicProfile
		if err := rows.Scan(&p.Username, &p.DisplayName, &p.Bio, &p.GitHubName, &p.BilibiliUID, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *sqliteService) GetPublicProfile(ctx context.Context, username string) (*PublicProfile, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT u.username,
		       COALESCE(p.display_name, ''),
		       COALESCE(p.bio, ''),
		       COALESCE(p.github_name, ''),
		       COALESCE(p.bilibili_uid, ''),
		       COALESCE(p.updated_at, u.created_at)
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.username = ? AND u.is_registered = 1
	`, username)
	var p PublicProfile
	if err := row.Scan(&p.Username, &p.DisplayName, &p.Bio, &p.GitHubName, &p.BilibiliUID, &p.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (s *sqliteService) UpdateMyProfile(ctx context.Context, userID string, update ProfileUpdate) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_profiles(user_id, display_name, bio, github_name, bilibili_uid, updated_at)
		VALUES(?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
		    display_name = excluded.display_name,
		    bio          = excluded.bio,
		    github_name  = excluded.github_name,
		    bilibili_uid = excluded.bilibili_uid,
		    updated_at   = excluded.updated_at
	`, userID, update.DisplayName, update.Bio, update.GitHubName, update.BilibiliUID, time.Now().UTC())
	return err
}
