package app

import (
	"context"
	"errors"
	"log"
	"net/http"
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
	Addr          string   // API 监听地址（HTTP_ADDR，默认 :8080）
	AdminAddr     string   // 后台 SPA 监听地址（HTTP_ADMIN_ADDR，默认 :8081；空字符串表示不启动）
	PublicAPIBase string   // 注入到 SPA index.html 的 API base URL，留空则按请求自动推导
	CORSOrigins   []string // 允许的跨域源
	AllowCreds    bool     // 是否允许携带凭据
	AllowHeaders  []string // 允许的自定义头
}

func loadConfig() Config {
	addr := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	if addr == "" {
		addr = ":8080"
	}
	adminAddr, adminAddrSet := os.LookupEnv("HTTP_ADMIN_ADDR")
	adminAddr = strings.TrimSpace(adminAddr)
	if !adminAddrSet {
		adminAddr = ":8081"
	}

	publicAPIBase := strings.TrimSpace(os.Getenv("HTTP_PUBLIC_API_BASE"))

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
	// 自动把 admin 端口加进 CORS 允许列表（localhost / 127.0.0.1 两种写法都加上）。
	for _, host := range []string{"http://localhost", "http://127.0.0.1"} {
		o := host + portOf(adminAddr)
		if !containsFold(origins, o) {
			origins = append(origins, o)
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
		Addr:          addr,
		AdminAddr:     adminAddr,
		PublicAPIBase: publicAPIBase,
		CORSOrigins:   origins,
		AllowCreds:    allowCreds,
		AllowHeaders:  allowHeaders,
	}
}

func Run(version, commit, build string) {
	cfg := loadConfig()

	if m := strings.TrimSpace(os.Getenv("GIN_MODE")); m != "" {
		gin.SetMode(m)
	}

	apiEngine := gin.New()
	apiEngine.Use(gin.Logger(), gin.Recovery())

	c := cors.DefaultConfig()
	c.AllowOrigins = cfg.CORSOrigins
	c.AllowCredentials = cfg.AllowCreds
	for _, h := range cfg.AllowHeaders {
		c.AddAllowHeaders(h)
	}
	c.MaxAge = 12 * time.Hour
	apiEngine.Use(cors.New(c))

	info := handler.NewInfoHandler(version, commit, build)
	apiEngine.GET("/status", info.HandleStatus)
	apiEngine.GET("/version", info.HandleVersion)

	// 模块（含各自的 /api/<mod>/...）。
	mod.MountAll(apiEngine)

	apiEngine.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message":   "Backend-Go is running.",
			"version":   version,
			"commit":    commit,
			"build":     build,
			"adminAddr": cfg.AdminAddr,
		})
	})

	// 后台 SPA：独立 engine + 独立端口。dist 未构建时返回 false，跳过启动。
	adminEngine, adminReady := adminui.BuildEngine(adminui.Options{
		PublicAPIBase: cfg.PublicAPIBase,
		MainAddr:      cfg.Addr,
	})

	apiSrv := &http.Server{Addr: cfg.Addr, Handler: apiEngine, ReadHeaderTimeout: 10 * time.Second}
	errCh := make(chan error, 2)

	go func() {
		log.Printf("[api] listening on %s", cfg.Addr)
		if err := apiSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	var adminSrv *http.Server
	if adminReady && cfg.AdminAddr != "" {
		adminSrv = &http.Server{Addr: cfg.AdminAddr, Handler: adminEngine, ReadHeaderTimeout: 10 * time.Second}
		go func() {
			log.Printf("[admin] SPA listening on %s", cfg.AdminAddr)
			if err := adminSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}()
	} else if !adminReady {
		log.Printf("[admin] SPA dist not built; admin port disabled. Run `cd web && pnpm build` to enable.")
	} else {
		log.Printf("[admin] HTTP_ADMIN_ADDR empty; admin port disabled")
	}

	if err := <-errCh; err != nil {
		// 关闭另一个，再退出。
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = apiSrv.Shutdown(ctx)
		if adminSrv != nil {
			_ = adminSrv.Shutdown(ctx)
		}
		log.Fatalf("服务器异常退出: %v", err)
	}
}

func portOf(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ":8081"
	}
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[i:]
	}
	return ":" + addr
}

func containsFold(list []string, v string) bool {
	for _, s := range list {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}
