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

// Attach 固定前缀（兼容老用法）：/api/avatar
func Attach(engine *gin.Engine) {
	envinit.Init()
	svc, err := NewServiceFromEnv()
	if err != nil {
		panic("avatar service init failed: " + err.Error())
	}

	// 静态资源：将 AVATAR_DIR 暴露到 AVATAR_URL_PREFIX
	engine.StaticFS(svc.URLPrefix, gin.Dir(svc.Dir, false))

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
	engine.StaticFS(svc.URLPrefix, gin.Dir(svc.Dir, false))

	if apiPrefix == "" {
		apiPrefix = "/api/avatar"
	}
	grp := engine.Group(apiPrefix)
	Mount(grp, svc)
}
