package envinit

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"

	"backend-go/pkg/paths"
)

const (
	dirName  = "config/rhythmgames"
	mainEnv  = ".env"
	localEnv = "local.env"
)

func defaultEnv() []byte {
	now := time.Now().Format(time.RFC3339)
	return []byte(
		"# Auto-generated on " + now + "\n" +
			"# Rhythm games module config.\n\n" +
			"# DX rating SVG cache TTL (seconds)\n" +
			"RHYTHMGAMES_CACHE_TTL_SECONDS=300\n" +
			"# Upstream HTTP timeout (seconds)\n" +
			"RHYTHMGAMES_HTTP_TIMEOUT_SECONDS=8\n",
	)
}

func Init() {
	base := paths.ExecDir()
	cfgDir := filepath.Join(base, dirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		log.Printf("[rhythmgames/envinit] mkdir %s: %v", cfgDir, err)
		return
	}
	envPath := filepath.Join(cfgDir, mainEnv)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if err := os.WriteFile(envPath, defaultEnv(), 0o644); err != nil {
			log.Printf("[rhythmgames/envinit] write default env: %v", err)
		} else {
			log.Printf("[rhythmgames/envinit] created %s", envPath)
		}
	}
	_ = godotenv.Load(envPath)
	_ = godotenv.Overload(filepath.Join(cfgDir, localEnv))
	log.Printf("[rhythmgames/envinit] loaded %s", cfgDir)
}
