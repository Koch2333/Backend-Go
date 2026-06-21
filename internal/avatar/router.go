package avatar

import (
	"backend-go/internal/avatar/envinit"
	"github.com/gin-gonic/gin"
)

func Mount(r *gin.RouterGroup, svc *Service) {
	h := NewHandler(svc)
	r.POST("", h.Upload)
	r.GET("/:id", h.Get)
}

// mountStaticOnce 把 dir 暴露到 prefix；如果该前缀的 GET 路由已经存在就跳过，
// 避免与其他模块（比如 aicweb）重复注册导致 gin 路由冲突 panic。
func mountStaticOnce(engine *gin.Engine, prefix, dir string) {
	wildcard := prefix + "/*filepath"
	for _, ri := range engine.Routes() {
		if ri.Method == "GET" && ri.Path == wildcard {
			return
		}
	}
	engine.StaticFS(prefix, gin.Dir(dir, false))
}

// Attach 固定前缀（兼容老用法）：/api/avatar
func Attach(engine *gin.Engine) {
	envinit.Init()
	svc, err := NewServiceFromEnv()
	if err != nil {
		panic("avatar service init failed: " + err.Error())
	}

	mountStaticOnce(engine, svc.URLPrefix, svc.Dir)

	grp := engine.Group("/api/avatar")
	Mount(grp, svc)
}

// AttachTo 自定义前缀 + 自动静态挂载
func AttachTo(engine *gin.Engine, apiPrefix string) {
	envinit.Init()
	svc, err := NewServiceFromEnv()
	if err != nil {
		panic("avatar service init failed: " + err.Error())
	}
	mountStaticOnce(engine, svc.URLPrefix, svc.Dir)

	if apiPrefix == "" {
		apiPrefix = "/api/avatar"
	}
	grp := engine.Group(apiPrefix)
	Mount(grp, svc)
}
