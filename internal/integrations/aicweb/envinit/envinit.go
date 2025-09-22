package envinit

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/joho/godotenv"
)

const Filename = ".env.development"

func defaultEnvContent() []byte {
	now := time.Now().Format(time.RFC3339)
	return []byte(
		"# Auto-generated on " + now + "\n" +
			"# 本文件仅用于本地/首次启动默认配置，可按需复制为 .env 或 .env.production\n\n" +
			"AICWEB_BASE_PREFIX=/api/aicweb\n" +
			"TURNSTILE_ENABLED=false\n" +
			"# TURNSTILE_SECRET=\n" +
			"GIN_MODE=debug\n" +
			"PORT=8080\n",
	)
}

func Init() {
	// 利用 runtime.Caller 确定当前源码所在目录
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Printf("[envinit] 无法获取当前文件路径")
		return
	}
	// thisFile 是 .../internal/integrations/aicweb/envinit/envinit.go
	// 所以我们取上级目录 aicweb
	aicwebDir := filepath.Dir(filepath.Dir(thisFile))

	envPath := filepath.Join(aicwebDir, Filename)

	// 如果不存在，就生成默认文件
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if err := os.WriteFile(envPath, defaultEnvContent(), 0o644); err != nil {
			log.Printf("[envinit] 创建默认环境文件失败: %v", err)
		} else {
			log.Printf("[envinit] 已创建默认环境文件: %s", envPath)
		}
	}

	// 尝试加载
	if err := godotenv.Load(envPath); err == nil {
		log.Printf("[envinit] 已加载环境文件: %s", envPath)
	} else {
		log.Printf("[envinit] 未加载环境文件: %v", err)
	}
}
