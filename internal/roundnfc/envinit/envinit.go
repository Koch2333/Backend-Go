package envinit

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

const (
	dirName  = "config/roundnfc"
	mainEnv  = ".env"
	localEnv = "local.env"
)

func defaultEnv() []byte {
	now := time.Now().Format(time.RFC3339)
	return []byte(
		"# Auto-generated on " + now + "\n" +
			"# RoundNFC module config.\n\n" +
			"ROUNDNFC_SQLITE_PATH=databases/roundnfc/roundnfc.db\n\n" +
			"# 对象存储 (Local driver)\n" +
			"ROUNDNFC_OBJECT_DIR=storage/roundnfc/objects\n" +
			"ROUNDNFC_OBJECT_HMAC_KEY=" + randHex(32) + "\n" +
			"ROUNDNFC_OBJECT_TTL_SECONDS=120\n" +
			"ROUNDNFC_MAX_UPLOAD_MB=8\n\n" +
			"# 风控\n" +
			"ROUNDNFC_TURNSTILE_SECRET=\n" +
			"ROUNDNFC_RATELIMIT_PER_MIN=12\n\n" +
			"# 后台账号（使用 cmd/genpw 生成 bcrypt 哈希；HASH 为空则后台禁用）\n" +
			"ROUNDNFC_ADMIN_USERNAME=admin\n" +
			"ROUNDNFC_ADMIN_PASSWORD_HASH=\n" +
			"ROUNDNFC_JWT_SECRET=" + randHex(32) + "\n" +
			"ROUNDNFC_JWT_TTL_HOURS=12\n\n" +
			"# TOTP (Google Authenticator)\n" +
			"ROUNDNFC_TOTP_ISSUER=RoundNFC\n\n" +
			"# WebAuthn / Passkey\n" +
			"# 生产环境请设置为实际域名，如 admin.example.com\n" +
			"ROUNDNFC_WEBAUTHN_RPID=localhost\n" +
			"ROUNDNFC_WEBAUTHN_RP_NAME=RoundNFC Admin\n" +
			"# 多个 origin 用逗号分隔\n" +
			"ROUNDNFC_WEBAUTHN_ORIGINS=http://localhost:5174\n",
	)
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func Init() {
	base, err := os.Getwd()
	if err != nil || base == "" {
		if exe, e := os.Executable(); e == nil {
			base = filepath.Dir(exe)
		}
	}
	if base == "" {
		log.Printf("[roundnfc/envinit] base dir not found; skip init")
		return
	}
	cfgDir := filepath.Join(base, dirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		log.Printf("[roundnfc/envinit] mkdir %s: %v", cfgDir, err)
		return
	}
	envPath := filepath.Join(cfgDir, mainEnv)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if err := os.WriteFile(envPath, defaultEnv(), 0o644); err != nil {
			log.Printf("[roundnfc/envinit] write default env: %v", err)
		} else {
			log.Printf("[roundnfc/envinit] created %s", envPath)
		}
	}
	_ = godotenv.Load(envPath)
	_ = godotenv.Overload(filepath.Join(cfgDir, localEnv))
	log.Printf("[roundnfc/envinit] loaded %s", cfgDir)
}
