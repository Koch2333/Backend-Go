package redirect

import (
	"backend-go/internal/redirect/envinit"
	"github.com/gin-gonic/gin"
)

func Mount(r *gin.RouterGroup) {
	svc, err := NewServiceFromEnv()
	if err != nil {
		panic("redirect service init failed: " + err.Error())
	}
	h := NewHandler(svc)
	r.GET("/:name", h.RedirectByName)
	r.GET("/pncs/:hwid", h.RedirectNFC)
}

// 兼容旧用法：固定 /api/redirect
func Attach(engine *gin.Engine) {
	envinit.Init()
	grp := engine.Group("/api/redirect")
	Mount(grp)
}

// 新增：可由外部决定前缀
func AttachTo(engine *gin.Engine, prefix string) {
	envinit.Init()
	if prefix == "" {
		prefix = "/api/redirect"
	}
	grp := engine.Group(prefix)
	Mount(grp)
}
