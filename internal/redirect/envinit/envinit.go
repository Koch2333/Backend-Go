package envinit

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

const (
	dirName  = "config/redirect"
	mainEnv  = ".env"
	localEnv = "local.env"
)

func defaultEnv() []byte {
	now := time.Now().Format(time.RFC3339)
	return []byte(
		"# Auto-generated on " + now + "\n" +
			"# Redirect module config.\n\n" +
			// ★ 默认数据库放在 ./databases/redirect/redirect.db
			"REDIRECT_SQLITE_PATH=databases/redirect/redirect.db\n" +
			"REDIRECT_NOT_FOUND_URL=https://koch2333.cn/404?name={name}\n" +
			"REDIRECT_NFC_REGISTERED_URL=https://koch2333.cn/pncs/ok?uid={userId}&hwid={hwid}\n" +
			"REDIRECT_NFC_UNREGISTERED_URL=https://koch2333.cn/pncs/register?hwid={hwid}\n",
	)
}

func Init() {
	base, err := os.Getwd()
	if err != nil || base == "" {
		if exe, e := os.Executable(); e == nil {
			base = filepath.Dir(exe)
		}
	}
	if base == "" {
		log.Printf("[redirect/envinit] base dir not found; skip init")
		return
	}

	cfgDir := filepath.Join(base, dirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		log.Printf("[redirect/envinit] mkdir %s: %v", cfgDir, err)
		return
	}

	envPath := filepath.Join(cfgDir, mainEnv)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if err := os.WriteFile(envPath, defaultEnv(), 0o644); err != nil {
			log.Printf("[redirect/envinit] write default env: %v", err)
		} else {
			log.Printf("[redirect/envinit] created %s", envPath)
		}
	}

	_ = godotenv.Load(envPath)
	_ = godotenv.Overload(filepath.Join(cfgDir, localEnv))
	log.Printf("[redirect/envinit] loaded %s", cfgDir)
}
