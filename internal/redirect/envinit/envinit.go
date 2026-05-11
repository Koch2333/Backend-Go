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
			"REDIRECT_NFC_UNREGISTERED_URL=https://koch2333.cn/pncs/register?hwid={hwid}\n\n" +
			"# 后台账号（使用 cmd/genpw 生成 bcrypt 哈希；HASH 为空则后台禁用）\n" +
			"REDIRECT_ADMIN_USERNAME=admin\n" +
			"REDIRECT_ADMIN_PASSWORD_HASH=\n" +
			"REDIRECT_JWT_SECRET=" + randHex(32) + "\n" +
			"REDIRECT_JWT_TTL_HOURS=12\n\n" +
			"# TOTP (Google Authenticator)\n" +
			"REDIRECT_TOTP_ISSUER=Redirect\n\n" +
			"# WebAuthn / Passkey\n" +
			"REDIRECT_WEBAUTHN_RPID=localhost\n" +
			"REDIRECT_WEBAUTHN_RP_NAME=Redirect Admin\n" +
			"REDIRECT_WEBAUTHN_ORIGINS=http://localhost:5174\n",
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
