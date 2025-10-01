package aicweb

import (
	"os"

	"backend-go/internal/integrations/aicweb/envinit"
	"github.com/gin-gonic/gin"
)

func Mount(r *gin.RouterGroup) {
	svc := NewServiceMemory()
	ts := NewTurnstileFromEnv()

	fs, err := NewFormServiceFromEnv()
	if err != nil {
		panic("failed to init sqlite form service: " + err.Error())
	}

	h := NewHandler(svc, ts, fs)

	r.POST("/user/register", h.Register)
	r.POST("/user/login", h.Login)

	prv := r.Group("", AuthRequired(svc))
	{
		prv.GET("/user/profile", h.Profile)
		prv.POST("/user/form", h.SubmitForm)
		prv.GET("/user/form", h.ListMyForms)
	}
}

// 兼容旧用法：从环境变量读前缀
func Attach(engine *gin.Engine) {
	envinit.Init()
	base := os.Getenv("AICWEB_BASE_PREFIX")
	if base == "" {
		base = "/api/aicweb"
	}
	grp := engine.Group(base)
	Mount(grp)
}

// 新增：可由外部决定前缀
func AttachTo(engine *gin.Engine, prefix string) {
	envinit.Init()
	if prefix == "" {
		prefix = "/api/aicweb"
	}
	grp := engine.Group(prefix)
	Mount(grp)
}
