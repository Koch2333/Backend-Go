package aicweb

import (
	"context"
	"database/sql"
	"io"
	"strings"
	"time"
)

// PublicProfile is the publicly visible profile of a user.
type PublicProfile struct {
	Username               string    `json:"username"`
	DisplayName            string    `json:"displayName"`
	Bio                    string    `json:"bio"`
	GitHubName             string    `json:"githubName"`
	BilibiliUID            string    `json:"bilibiliUid"`
	MessageToSchool        string    `json:"messageToSchool"`
	MessageToUnderclassmen string    `json:"messageToUnderclassmen"`
	AvatarUrl              string    `json:"avatarUrl"`
	BannerUrl              string    `json:"bannerUrl"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

// ProfileUpdate is the writable portion of a user's profile.
type ProfileUpdate struct {
	DisplayName            string `json:"displayName"`
	Bio                    string `json:"bio"`
	GitHubName             string `json:"githubName"`
	BilibiliUID            string `json:"bilibiliUid"`
	MessageToSchool        string `json:"messageToSchool"`
	MessageToUnderclassmen string `json:"messageToUnderclassmen"`
	AvatarUrl              string `json:"avatarUrl"`
	BannerUrl              string `json:"bannerUrl"`
}

// MediaUploader processes an image stream and returns its public URL.
type MediaUploader interface {
	Upload(r io.Reader) (publicURL string, err error)
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
    user_id                  TEXT PRIMARY KEY,
    display_name             TEXT NOT NULL DEFAULT '',
    bio                      TEXT NOT NULL DEFAULT '',
    github_name              TEXT NOT NULL DEFAULT '',
    bilibili_uid             TEXT NOT NULL DEFAULT '',
    message_to_school        TEXT NOT NULL DEFAULT '',
    message_to_underclassmen TEXT NOT NULL DEFAULT '',
    avatar_url               TEXT NOT NULL DEFAULT '',
    banner_url               TEXT NOT NULL DEFAULT '',
    updated_at               DATETIME NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_user_profiles_uid ON user_profiles(user_id);
`

func migrateProfiles(db *sql.DB) error {
	_, err := db.Exec(profileDDL)
	return err
}

// migrateProfileColumns adds columns introduced after the initial schema.
// SQLite has no ADD COLUMN IF NOT EXISTS, so we ignore "duplicate column" errors.
func migrateProfileColumns(db *sql.DB) error {
	alters := []string{
		`ALTER TABLE user_profiles ADD COLUMN message_to_school TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE user_profiles ADD COLUMN message_to_underclassmen TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE user_profiles ADD COLUMN avatar_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE user_profiles ADD COLUMN banner_url TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range alters {
		if _, err := db.Exec(stmt); err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
				return err
			}
		}
	}
	return nil
}

// parseDBTime tries to parse a datetime string stored by modernc.org/sqlite.
func parseDBTime(s string) time.Time {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// profileFromExisting returns a ProfileUpdate that preserves existing fields,
// used when doing partial updates (e.g. only updating avatarUrl).
func profileFromExisting(p *PublicProfile) ProfileUpdate {
	if p == nil {
		return ProfileUpdate{}
	}
	return ProfileUpdate{
		DisplayName:            p.DisplayName,
		Bio:                    p.Bio,
		GitHubName:             p.GitHubName,
		BilibiliUID:            p.BilibiliUID,
		MessageToSchool:        p.MessageToSchool,
		MessageToUnderclassmen: p.MessageToUnderclassmen,
		AvatarUrl:              p.AvatarUrl,
		BannerUrl:              p.BannerUrl,
	}
}

func (s *sqliteService) ListPublicProfiles(ctx context.Context) ([]PublicProfile, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.username,
		       COALESCE(p.display_name, ''),
		       COALESCE(p.bio, ''),
		       COALESCE(p.github_name, ''),
		       COALESCE(p.bilibili_uid, ''),
		       COALESCE(p.message_to_school, ''),
		       COALESCE(p.message_to_underclassmen, ''),
		       COALESCE(p.avatar_url, ''),
		       COALESCE(p.banner_url, ''),
		       CAST(COALESCE(p.updated_at, u.created_at) AS TEXT)
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
		var updatedAt string
		if err := rows.Scan(
			&p.Username, &p.DisplayName, &p.Bio, &p.GitHubName, &p.BilibiliUID,
			&p.MessageToSchool, &p.MessageToUnderclassmen,
			&p.AvatarUrl, &p.BannerUrl, &updatedAt,
		); err != nil {
			return nil, err
		}
		p.UpdatedAt = parseDBTime(updatedAt)
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
		       COALESCE(p.message_to_school, ''),
		       COALESCE(p.message_to_underclassmen, ''),
		       COALESCE(p.avatar_url, ''),
		       COALESCE(p.banner_url, ''),
		       CAST(COALESCE(p.updated_at, u.created_at) AS TEXT)
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.username = ? AND u.is_registered = 1
	`, username)
	var p PublicProfile
	var updatedAt string
	if err := row.Scan(
		&p.Username, &p.DisplayName, &p.Bio, &p.GitHubName, &p.BilibiliUID,
		&p.MessageToSchool, &p.MessageToUnderclassmen,
		&p.AvatarUrl, &p.BannerUrl, &updatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	p.UpdatedAt = parseDBTime(updatedAt)
	return &p, nil
}

func (s *sqliteService) UpdateMyProfile(ctx context.Context, userID string, update ProfileUpdate) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_profiles(
		    user_id, display_name, bio, github_name, bilibili_uid,
		    message_to_school, message_to_underclassmen,
		    avatar_url, banner_url, updated_at
		)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
		    display_name             = excluded.display_name,
		    bio                      = excluded.bio,
		    github_name              = excluded.github_name,
		    bilibili_uid             = excluded.bilibili_uid,
		    message_to_school        = excluded.message_to_school,
		    message_to_underclassmen = excluded.message_to_underclassmen,
		    avatar_url               = excluded.avatar_url,
		    banner_url               = excluded.banner_url,
		    updated_at               = excluded.updated_at
	`, userID,
		update.DisplayName, update.Bio, update.GitHubName, update.BilibiliUID,
		update.MessageToSchool, update.MessageToUnderclassmen,
		update.AvatarUrl, update.BannerUrl, time.Now().UTC())
	return err
}
