package aicweb

import (
	"os"

	"github.com/gin-gonic/gin"
)

func Mount(r *gin.RouterGroup) {
	svc := NewServiceMemory()
	ts := NewTurnstileFromEnv() // <- 新增：根据环境变量创建校验器
	h := NewHandler(svc, ts)

	r.POST("/user/register", h.Register)
	r.POST("/user/login", h.Login)

	prv := r.Group("", AuthRequired(svc))
	{
		prv.GET("/user/profile", h.Profile)
	}
}

func Attach(engine *gin.Engine) {
	base := os.Getenv("AICWEB_BASE_PREFIX")
	if base == "" {
		base = "/api/aicweb"
	}
	grp := engine.Group(base)
	Mount(grp)
}
