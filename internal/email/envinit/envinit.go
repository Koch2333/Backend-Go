package envinit

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

const (
	dirName  = "config/email"
	mainEnv  = ".env"
	localEnv = "local.env"
)

func defaultEnv() []byte {
	now := time.Now().Format(time.RFC3339)
	return []byte(
		"# Auto-generated on " + now + "\n" +
			"# Email module config\n\n" +
			// 策略：smtp | log | none
			"EMAIL_STRATEGY=smtp\n" +
			"\n# SMTP settings\n" +
			"SMTP_HOST=smtp.example.com\n" +
			"SMTP_PORT=587\n" +
			"SMTP_USERNAME=no-reply@example.com\n" +
			"SMTP_PASSWORD=your-password\n" +
			"SMTP_FROM=No Reply <no-reply@example.com>\n",
	)
}

func Init() {
	base, _ := os.Getwd()
	if base == "" {
		if exe, err := os.Executable(); err == nil {
			base = filepath.Dir(exe)
		}
	}
	if base == "" {
		log.Printf("[email/envinit] base dir not found; skip init")
		return
	}
	cfg := filepath.Join(base, dirName)
	_ = os.MkdirAll(cfg, 0o755)

	envPath := filepath.Join(cfg, mainEnv)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		_ = os.WriteFile(envPath, defaultEnv(), 0o644)
	}
	_ = godotenv.Load(envPath)
	_ = godotenv.Overload(filepath.Join(cfg, localEnv))
	log.Printf("[email/envinit] loaded %s", cfg)
}
