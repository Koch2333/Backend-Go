package aicweb

import (
	"os"

	"backend-go/internal/integrations/aicweb/envinit"
	"github.com/gin-gonic/gin"
)

func Mount(r *gin.RouterGroup) {
	svc := NewServiceMemory()
	ts := NewTurnstileFromEnv()

	// SQLite 表单服务
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

		// 表单接口（受保护）
		prv.POST("/user/form", h.SubmitForm)
		prv.GET("/user/form", h.ListMyForms)
	}
}

func Attach(engine *gin.Engine) {
	// 生成并加载 internal/integrations/aicweb/.env.development
	envinit.Init()

	base := os.Getenv("AICWEB_BASE_PREFIX")
	if base == "" {
		base = "/api/aicweb"
	}
	grp := engine.Group(base)
	Mount(grp)
}
