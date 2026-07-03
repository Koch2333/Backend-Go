// cmd/roundnfc 是「只包含 RoundNFC 模块」的 Standalone 入口。
// 与 cmd/server 不同，这里不走 internal/bootstrap/mod 的 autogen 扫描，
// 避免把其他业务模块一起链接进他人打包的二进制。
package main

import (
	"log"
	"os"
	"strings"
	"time"

	"backend-go/internal/handler"
	"backend-go/internal/roundnfc"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	Version = "dev"
	Commit  = "none"
	Build   = "local"
)

func main() {
	addr := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	if addr == "" {
		addr = ":8080"
	}
	if m := strings.TrimSpace(os.Getenv("GIN_MODE")); m != "" {
		gin.SetMode(m)
	}
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	c := cors.DefaultConfig()
	if v := strings.TrimSpace(os.Getenv("HTTP_CORS_ORIGINS")); v != "" {
		var origins []string
		for _, p := range strings.Split(v, ",") {
			if p = strings.TrimSpace(p); p != "" {
				origins = append(origins, p)
			}
		}
		c.AllowOrigins = origins
	} else {
		c.AllowOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	}
	c.AllowCredentials = true
	c.AddAllowHeaders("Authorization", "Content-Type", "CF-Turnstile-Response", "X-App-Token")
	c.MaxAge = 12 * time.Hour
	engine.Use(cors.New(c))

	info := handler.NewInfoHandler(Version, Commit, Build)
	engine.GET("/status", info.HandleStatus)
	engine.GET("/version", info.HandleVersion)

	prefix := strings.TrimSpace(os.Getenv("ROUNDNFC_PREFIX"))
	if prefix == "" {
		prefix = "/api/roundnfc"
	}
	if err := roundnfc.AttachTo(engine, prefix); err != nil {
		log.Fatalf("attach roundnfc: %v", err)
	}

	log.Printf("RoundNFC standalone listening on %s (prefix=%s)", addr, prefix)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
