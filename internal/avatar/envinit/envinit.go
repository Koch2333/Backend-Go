package envinit

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

const (
	dirName  = "config/avatar"
	mainEnv  = ".env"      // 模块主配置
	localEnv = "local.env" // 开发者本地覆盖
)

func defaultEnv() []byte {
	now := time.Now().Format(time.RFC3339)
	return []byte(
		"# Auto-generated on " + now + "\n" +
			"# Avatar module config.\n\n" +
			// 文件保存目录（相对运行目录）
			"AVATAR_DIR=assets/avatar\n" +
			// 返回 URL 的前缀（会被静态路由挂载）
			"AVATAR_URL_PREFIX=/assets/avatar\n" +
			// 上传单文件大小上限（MB）
			"AVATAR_MAX_MB=5\n" +
			// WebP 质量（0-100）
			"AVATAR_WEBP_QUALITY=80\n",
	)
}

// Init: 在工作目录（失败回退可执行目录）创建 config/avatar/.env 并加载
func Init() {
	base, err := os.Getwd()
	if err != nil || base == "" {
		if exe, e := os.Executable(); e == nil {
			base = filepath.Dir(exe)
		}
	}
	if base == "" {
		log.Printf("[avatar/envinit] base dir not found; skip init")
		return
	}

	cfgDir := filepath.Join(base, dirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		log.Printf("[avatar/envinit] mkdir %s: %v", cfgDir, err)
		return
	}

	envPath := filepath.Join(cfgDir, mainEnv)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if err := os.WriteFile(envPath, defaultEnv(), 0o644); err != nil {
			log.Printf("[avatar/envinit] write default env: %v", err)
		} else {
			log.Printf("[avatar/envinit] created %s", envPath)
		}
	}

	_ = godotenv.Load(envPath)
	_ = godotenv.Overload(filepath.Join(cfgDir, localEnv))
	log.Printf("[avatar/envinit] loaded %s", cfgDir)
}
