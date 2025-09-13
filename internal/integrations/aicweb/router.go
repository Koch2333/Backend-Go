package aicweb

import (
	"os"

	"github.com/gin-gonic/gin"
)

// Mount 把路由注册到传入的 RouterGroup（便于外部控制前缀与中间件）
func Mount(r *gin.RouterGroup) {
	svc := NewServiceMemory()
	h := NewHandler(svc)

	r.POST("/user/register", h.Register)
	r.POST("/user/login", h.Login)

	// 受保护示例
	prv := r.Group("", AuthRequired(svc))
	{
		prv.GET("/user/profile", h.Profile)
	}
}

// Attach 便捷函数：直接在 *gin.Engine 上挂载，基路径可通过 env 配置
func Attach(engine *gin.Engine) {
	base := os.Getenv("AICWEB_BASE_PREFIX")
	if base == "" {
		base = "/api/aicweb"
	}
	grp := engine.Group(base)
	Mount(grp)
}
