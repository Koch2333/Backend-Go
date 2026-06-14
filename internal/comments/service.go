package comments

import (
	"fmt"
	"os"
)

type Config struct {
	SQLitePath string
	JWTSecret  []byte
}

type Service struct {
	cfg   Config
	store *Store
}

func NewServiceFromEnv() (*Service, error) {
	cfg := Config{
		SQLitePath: envOr("COMMENTS_SQLITE_PATH", "databases/comments/comments.db"),
		JWTSecret:  []byte(os.Getenv("COMMENTS_JWT_SECRET")),
	}
	if len(cfg.JWTSecret) == 0 {
		cfg.JWTSecret = []byte(os.Getenv("ROUNDNFC_JWT_SECRET"))
	}
	store, err := openStore(cfg.SQLitePath)
	if err != nil {
		return nil, fmt.Errorf("comments: open store: %w", err)
	}
	return &Service{cfg: cfg, store: store}, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
