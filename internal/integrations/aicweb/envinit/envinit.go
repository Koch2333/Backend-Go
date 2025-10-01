package envinit

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

const (
	dirName  = "config/aicweb"
	mainEnv  = ".env"      // 模块主配置
	localEnv = "local.env" // 本地覆盖（gitignore）
)

func defaultEnv() []byte {
	now := time.Now().Format(time.RFC3339)
	return []byte(
		"# Auto-generated on " + now + "\n" +
			"# AICWeb module config. Production should set real env vars.\n\n" +
			"AICWEB_BASE_PREFIX=/api/aicweb\n" +
			"TURNSTILE_ENABLED=false\n" +
			"# TURNSTILE_SECRET=\n" +
			// ★ 默认数据库放在 ./databases/aicweb/forms.db
			"AICWEB_SQLITE_PATH=databases/aicweb/forms.db\n",
	)
}

// 在工作目录（失败则退回可执行目录）创建 config/aicweb/.env，并加载
func Init() {
	base, err := os.Getwd()
	if err != nil || base == "" {
		if exe, e := os.Executable(); e == nil {
			base = filepath.Dir(exe)
		}
	}
	if base == "" {
		log.Printf("[aicweb/envinit] base dir not found; skip init")
		return
	}

	cfgDir := filepath.Join(base, dirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		log.Printf("[aicweb/envinit] mkdir %s: %v", cfgDir, err)
		return
	}

	envPath := filepath.Join(cfgDir, mainEnv)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if err := os.WriteFile(envPath, defaultEnv(), 0o644); err != nil {
			log.Printf("[aicweb/envinit] write default env: %v", err)
		} else {
			log.Printf("[aicweb/envinit] created %s", envPath)
		}
	}

	_ = godotenv.Load(envPath)
	_ = godotenv.Overload(filepath.Join(cfgDir, localEnv))
	log.Printf("[aicweb/envinit] loaded %s", cfgDir)
}
