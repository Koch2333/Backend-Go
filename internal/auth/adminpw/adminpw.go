// Package adminpw 解析后台管理员凭据。
//
// 优先级：
//  1. <MOD>_ADMIN_PASSWORD_HASH 已设置：直接当 bcrypt hash 用（最安全，适合公网）。
//  2. <MOD>_ADMIN_PASSWORD 已设置：启动时 bcrypt 一次，存内存里用（最省事，
//     用户不用跑 cmd/genpw）。
//  3. 都没设置：返回空字符串，调用方据此判定后台禁用。
//
// 仅在两个 env 都被显式设置时，HASH 胜出（更安全）。
package adminpw

import (
	"log"
	"os"
	"strings"

	"backend-go/internal/auth"
)

// Resolve 返回可以嗂给 auth.VerifyPassword 的 bcrypt hash 字符串。
// modTag 用于日志（例如 "redirect" / "roundnfc"），envPrefix 是 "REDIRECT" / "ROUNDNFC"。
// 返回空字符串说明既没 hash 也没 plain password，调用方应禁用后台登录。
func Resolve(modTag, envPrefix string) string {
	hash := strings.TrimSpace(os.Getenv(envPrefix + "_ADMIN_PASSWORD_HASH"))
	if hash != "" {
		return hash
	}
	plain := os.Getenv(envPrefix + "_ADMIN_PASSWORD")
	if plain == "" {
		return ""
	}
	h, err := auth.HashPassword(plain)
	if err != nil {
		log.Printf("[%s/adminpw] bcrypt plain password failed: %v", modTag, err)
		return ""
	}
	log.Printf("[%s/adminpw] using %s_ADMIN_PASSWORD (hashed in memory; set %s_ADMIN_PASSWORD_HASH for prod)", modTag, envPrefix, envPrefix)
	return h
}
