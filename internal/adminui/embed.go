// Package adminui 内嵌 web/dist 构建产物，在 /admin 路径下供应管理后台 SPA。
//
// 使用方式：
//	cd web && pnpm install && pnpm build      # 输出到 internal/adminui/dist
//	go build ./cmd/server                     # 二进制里就含后台了
//
// 未构建时（只有 .gitkeep）会静默不挂，并不影响 Go 二进制编译。
package adminui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var distFS embed.FS

// Mount serves the built admin SPA at the given prefix (default "/admin").
func Mount(engine *gin.Engine, prefix string) {
	if strings.TrimSpace(prefix) == "" {
		prefix = "/admin"
	}
	prefix = "/" + strings.Trim(prefix, "/")

	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return
	}
	indexBytes, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		// dist 里还只有 .gitkeep，或者 web/ 还没 build。静默不挂。
		return
	}

	fileSrv := http.FileServer(http.FS(sub))
	engine.GET(prefix, func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexBytes)
	})
	engine.GET(prefix+"/*path", func(c *gin.Context) {
		p := strings.TrimPrefix(c.Param("path"), "/")
		if p == "" {
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexBytes)
			return
		}
		if _, err := fs.Stat(sub, p); err != nil {
			// SPA fallback：未知路径还 index.html，路由交给前端。
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexBytes)
			return
		}
		http.StripPrefix(prefix, fileSrv).ServeHTTP(c.Writer, c.Request)
	})
}
