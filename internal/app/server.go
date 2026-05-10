package app

import (
	"log"
	"os"
	"strings"
	"time"

	"backend-go/internal/adminui"
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
	AdminPrefix  string   // 后台 SPA 挂载路径（默认 /admin）
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
		origins = []string{
			"http://localhost:5173",
			"http://localhost:5174",
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
	adminPrefix := strings.TrimSpace(os.Getenv("ADMIN_UI_PREFIX"))
	if adminPrefix == "" {
		adminPrefix = "/admin"
	}
	return Config{
		Addr:         addr,
		CORSOrigins:  origins,
		AllowCreds:   allowCreds,
		AllowHeaders: allowHeaders,
		AdminPrefix:  adminPrefix,
	}
}

func Run(version, commit, build string) {
	cfg := loadConfig()

	if m := strings.TrimSpace(os.Getenv("GIN_MODE")); m != "" {
		gin.SetMode(m)
	}

	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	c := cors.DefaultConfig()
	c.AllowOrigins = cfg.CORSOrigins
	c.AllowCredentials = cfg.AllowCreds
	for _, h := range cfg.AllowHeaders {
		c.AddAllowHeaders(h)
	}
	c.MaxAge = 12 * time.Hour
	engine.Use(cors.New(c))

	info := handler.NewInfoHandler(version, commit, build)
	engine.GET("/status", info.HandleStatus)
	engine.GET("/version", info.HandleVersion)

	// 自动挂载所有已注册模块（依赖 internal/bootstrap/mod/autogen_imports.go）
	mod.MountAll(engine)

	// 内嵌后台 SPA（web/）。未构建时静默不挂。
	adminui.Mount(engine, cfg.AdminPrefix)

	engine.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "Backend-Go is running.",
			"version": version,
			"commit":  commit,
			"build":   build,
		})
	})

	log.Printf("服务器启动于 %s (admin=%s)", cfg.Addr, cfg.AdminPrefix)
	if err := engine.Run(cfg.Addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
