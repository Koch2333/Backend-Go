package envinit

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"

	"backend-go/pkg/paths"
)

const (
	dirName  = "config/comments"
	mainEnv  = ".env"
	localEnv = "local.env"
)

func defaultEnv() []byte {
	now := time.Now().Format(time.RFC3339)
	return []byte(
		"# Auto-generated on " + now + "\n" +
			"# Comments module config.\n\n" +
			"COMMENTS_SQLITE_PATH=databases/comments/comments.db\n\n" +
			"# JWT secret for admin endpoints (falls back to ROUNDNFC_JWT_SECRET if empty)\n" +
			"COMMENTS_JWT_SECRET=" + randHex(32) + "\n",
	)
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func Init() {
	base := paths.ExecDir()
	cfgDir := filepath.Join(base, dirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		log.Printf("[comments/envinit] mkdir %s: %v", cfgDir, err)
		return
	}
	envPath := filepath.Join(cfgDir, mainEnv)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if err := os.WriteFile(envPath, defaultEnv(), 0o644); err != nil {
			log.Printf("[comments/envinit] write default env: %v", err)
		} else {
			log.Printf("[comments/envinit] created %s", envPath)
		}
	}
	_ = godotenv.Load(envPath)
	_ = godotenv.Overload(filepath.Join(cfgDir, localEnv))
	log.Printf("[comments/envinit] loaded %s", cfgDir)
}
