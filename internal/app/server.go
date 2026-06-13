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
	Addr            string   // 监听地址（默认 :8080）
	CORSOrigins     []string // 允许的跨域源
	AllowAllOrigins bool     // 允许所有源（未配置 HTTP_CORS_ORIGINS 时默认开启）
	AllowCreds      bool     // 是否允许携带凭据
	AllowHeaders    []string // 允许的自定义头
	AdminPrefix     string   // 后台 SPA 挂载路径（默认 /admin）
}

func loadConfig() Config {
	addr := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	if addr == "" {
		addr = ":8080"
	}
	var origins []string
	allowAll := false
	if v := strings.TrimSpace(os.Getenv("HTTP_CORS_ORIGINS")); v == "*" {
		allowAll = true
	} else if v != "" {
		for _, p := range strings.Split(v, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				origins = append(origins, p)
			}
		}
	} else {
		allowAll = true
		log.Println("[cors] HTTP_CORS_ORIGINS not set, allowing all origins. Set it in production!")
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
		Addr:            addr,
		CORSOrigins:     origins,
		AllowAllOrigins: allowAll,
		AllowCreds:      allowCreds,
		AllowHeaders:    allowHeaders,
		AdminPrefix:     adminPrefix,
	}
}

func Run(version, commit, build string) {
	cfg := loadConfig()

	if m := strings.TrimSpace(os.Getenv("GIN_MODE")); m != "" {
		gin.SetMode(m)
	}

	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	corsCfg := cors.DefaultConfig()
	if cfg.AllowAllOrigins {
		corsCfg.AllowOriginFunc = func(_ string) bool { return true }
	} else {
		corsCfg.AllowOrigins = cfg.CORSOrigins
	}
	corsCfg.AllowCredentials = cfg.AllowCreds
	for _, h := range cfg.AllowHeaders {
		corsCfg.AddAllowHeaders(h)
	}
	corsCfg.MaxAge = 12 * time.Hour
	engine.Use(cors.New(corsCfg))

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
