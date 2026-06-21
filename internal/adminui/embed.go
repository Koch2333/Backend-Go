// Package adminui 内嵌 web/dist 构建产物，为后台 SPA 提供专用 HTTP 服务。
//
// 用法：通过 BuildEngine() 拿到一个只服务 SPA 的 *gin.Engine，
// 由 app 层挂到独立的端口（HTTP_ADMIN_ADDR）。SPA 自身向主端口的
// 后端 API 发起请求，cross-origin 由 CORS 处理。
//
// 未构建（dist 里只有 .gitkeep）时 BuildEngine 返回 nil, false，
// app 层据此跳过启动 admin 端口。
package adminui

import (
	"bytes"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var distFS embed.FS

// runtimeMarker 是注入运行时配置的占位标记。我们把它塞到 <head> 末尾，
// 每次请求 index.html 时再按当前请求把 API base URL 替换进去。
const runtimeMarker = "</head>"

// Options 控制 SPA 服务行为。
type Options struct {
	// PublicAPIBase 显式指定 SPA 调用 API 的 base URL（含 scheme + host + port）。
	// 留空时按请求 Host 自动推导（替换端口为 MainAddr 解析出的端口）。
	PublicAPIBase string
	// MainAddr 是主 API 端口的监听地址（如 ":8080"）。仅在 PublicAPIBase 为空
	// 时用于自动推导。
	MainAddr string
}

// BuildEngine 构造一个只服务 SPA 的 gin.Engine。dist 未构建时返回 nil, false。
func BuildEngine(opts Options) (*gin.Engine, bool) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, false
	}
	indexBytes, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		return nil, false
	}

	// 在 </head> 之前插入一个空的 <script> 占位（实际脚本体每次请求生成）。
	// 这里只是把 indexBytes 切成「头/尾」两段，避免每次重新搜索字符串。
	headIdx := bytes.LastIndex(indexBytes, []byte(runtimeMarker))
	var head, tail []byte
	if headIdx >= 0 {
		head = indexBytes[:headIdx]
		tail = indexBytes[headIdx:]
	} else {
		head = indexBytes
		tail = nil
	}

	mainPort := portOf(opts.MainAddr)

	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	fileSrv := http.FileServer(http.FS(sub))

	serveIndex := func(c *gin.Context) {
		apiBase := resolveAPIBase(c, opts.PublicAPIBase, mainPort)
		script := buildRuntimeScript(apiBase)
		var buf bytes.Buffer
		buf.Grow(len(head) + len(script) + len(tail))
		buf.Write(head)
		buf.WriteString(script)
		buf.Write(tail)
		c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
	}

	// 静态资源：vite 产物默认输出到 /assets/...
	engine.GET("/assets/*filepath", func(c *gin.Context) {
		fileSrv.ServeHTTP(c.Writer, c.Request)
	})
	// 根目录的常见静态文件（favicon 等）。fs.Stat 不通过就 fallback 给 SPA。
	engine.GET("/favicon.ico", func(c *gin.Context) {
		if _, err := fs.Stat(sub, "favicon.ico"); err == nil {
			fileSrv.ServeHTTP(c.Writer, c.Request)
			return
		}
		c.Status(http.StatusNotFound)
	})

	engine.GET("/", serveIndex)
	// SPA 客户端路由 fallback：所有未命中具体路由的 GET 都回 index.html。
	// /api/* 永远不能 fallback —— admin 端口没有 API，必须明确返 404，
	// 否则前端会把 HTML 当 JSON 解析出错误。
	engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "this port serves the admin SPA only; API lives on HTTP_ADDR"})
			return
		}
		if c.Request.Method != http.MethodGet {
			c.Status(http.StatusMethodNotAllowed)
			return
		}
		serveIndex(c)
	})

	log.Printf("[adminui] SPA engine ready (main api derived: port=%s public=%q)", mainPort, opts.PublicAPIBase)
	return engine, true
}

// buildRuntimeScript 生成 <script>window.__ROAST_RUNTIME = {...}</script>，
// 用 encoding/json 编码 apiBase 以避免脚本注入风险。
func buildRuntimeScript(apiBase string) string {
	payload := map[string]string{"apiBase": apiBase}
	b, _ := json.Marshal(payload)
	return `<script>window.__ROAST_RUNTIME=` + string(b) + `;</script>`
}

// resolveAPIBase 决定要注入的 API base URL：
//  1. PublicAPIBase 优先
//  2. 否则用 scheme://<request-host-without-port>:<mainPort>
func resolveAPIBase(c *gin.Context, publicAPIBase, mainPort string) string {
	if publicAPIBase != "" {
		return strings.TrimRight(publicAPIBase, "/")
	}
	host := c.Request.Host
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	if host == "" {
		host = "localhost"
	}
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + host + mainPort
}

// portOf 把 ":8080" / "0.0.0.0:8080" 提取成 ":8080"。
func portOf(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ":8080"
	}
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[i:]
	}
	return ":" + addr
}
