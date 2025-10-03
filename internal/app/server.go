package app

import (
	"log"
	"os"
	"strings"
	"time"

	"backend-go/internal/bootstrap/mod"
	"backend-go/internal/handler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Config struct {
	Addr         string   // 监听地址（默认 :8080）
	CORSOrigins  []string // 允许的跨域源
	AllowCreds   bool     // 是否允许携带凭据
	AllowHeaders []string // 允许的自定义头
}

func loadConfig() Config {
	addr := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	if addr == "" {
		addr = ":8080"
	}
	var origins []string
	if v := strings.TrimSpace(os.Getenv("HTTP_CORS_ORIGINS")); v != "" {
		for _, p := range strings.Split(v, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				origins = append(origins, p)
			}
		}
	} else {
		// 默认常见本地/示例域名
		origins = []string{
			"http://localhost:5173",
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"https://koch2333.cn",
		}
	}
	allowCreds := true
	if v := strings.TrimSpace(os.Getenv("HTTP_CORS_CREDENTIALS")); v != "" {
		allowCreds = strings.EqualFold(v, "true") || v == "1" || strings.EqualFold(v, "yes")
	}
	allowHeaders := []string{"CF-Turnstile-Response", "Authorization", "Content-Type"}
	if v := strings.TrimSpace(os.Getenv("HTTP_CORS_HEADERS")); v != "" {
		allowHeaders = nil
		for _, h := range strings.Split(v, ",") {
			h = strings.TrimSpace(h)
			if h != "" {
				allowHeaders = append(allowHeaders, h)
			}
		}
	}
	return Config{
		Addr:         addr,
		CORSOrigins:  origins,
		AllowCreds:   allowCreds,
		AllowHeaders: allowHeaders,
	}
}

func Run(version, commit, build string) {
	cfg := loadConfig()

	// Gin 模式：默认为 debug；生产可设 GIN_MODE=release
	if m := strings.TrimSpace(os.Getenv("GIN_MODE")); m != "" {
		gin.SetMode(m)
	}

	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	// CORS
	c := cors.DefaultConfig()
	c.AllowOrigins = cfg.CORSOrigins
	c.AllowCredentials = cfg.AllowCreds
	for _, h := range cfg.AllowHeaders {
		c.AddAllowHeaders(h)
	}
	// 预检缓存
	c.MaxAge = 12 * time.Hour
	engine.Use(cors.New(c))

	// 健康与版本
	info := handler.NewInfoHandler(version, commit, build)
	engine.GET("/status", info.HandleStatus)
	engine.GET("/version", info.HandleVersion)

	// ★ 自动挂载所有模块（依赖 autogen_imports.go 中的空导入触发各模块 init() 注册）
	mod.MountAll(engine)

	// 根路由
	engine.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "Backend-Go is running.",
			"version": version,
			"commit":  commit,
			"build":   build,
		})
	})

	log.Printf("服务器启动于 %s", cfg.Addr)
	if err := engine.Run(cfg.Addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
