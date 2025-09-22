package aicweb

import (
	"os"

	"backend-go/internal/integrations/aicweb/envinit"
	"github.com/gin-gonic/gin"
)

func Mount(r *gin.RouterGroup) {
	svc := NewServiceMemory()
	ts := NewTurnstileFromEnv()
	h := NewHandler(svc, ts)

	r.POST("/user/register", h.Register)
	r.POST("/user/login", h.Login)

	prv := r.Group("", AuthRequired(svc))
	{
		prv.GET("/user/profile", h.Profile)
	}
}

func Attach(engine *gin.Engine) {
	// 先初始化/加载 aicweb 目录下的 .env.development
	envinit.Init()

	base := os.Getenv("AICWEB_BASE_PREFIX")
	if base == "" {
		base = "/api/aicweb"
	}
	grp := engine.Group(base)
	Mount(grp)
}
