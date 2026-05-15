package redirect

import (
	"backend-go/internal/auth"
	"backend-go/internal/authflow"
	"backend-go/internal/redirect/envinit"

	"github.com/gin-gonic/gin"
)

func Mount(r *gin.RouterGroup, svc *Service) {
	pub := NewHandler(svc)
	adm := newAdminHandler(svc)
	flow := authflow.New(svc.AuthFlowConfig())

	// admin first so static "/admin" segment wins over wildcard "/:name"
	admin := r.Group("/admin")
	flow.Mount(admin)

	authed := admin.Group("", auth.Required(svc.Admin.JWTSecret))
	authed.GET("/rules", adm.ListRules)
	authed.POST("/rules", adm.UpsertRule)
	authed.PUT("/rules/:name", adm.UpsertRule)
	authed.DELETE("/rules/:name", adm.DeleteRule)
	authed.GET("/cards", adm.ListCards)
	authed.POST("/cards", adm.UpsertCard)
	authed.PUT("/cards/:hwid", adm.UpsertCard)
	authed.DELETE("/cards/:hwid", adm.DeleteCard)

	// public — fixed segments first, then the wildcard catch-all
	r.GET("/pncs/:hwid", pub.RedirectNFC)
	r.GET("/:name", pub.RedirectByName)
}

// 兼容旧用法：固定 /api/redirect
func Attach(engine *gin.Engine) {
	envinit.Init()
	svc, err := NewServiceFromEnv()
	if err != nil {
		panic("redirect service init failed: " + err.Error())
	}
	Mount(engine.Group("/api/redirect"), svc)
}

// 新增：可由外部决定前缀
func AttachTo(engine *gin.Engine, prefix string) {
	envinit.Init()
	if prefix == "" {
		prefix = "/api/redirect"
	}
	svc, err := NewServiceFromEnv()
	if err != nil {
		panic("redirect service init failed: " + err.Error())
	}
	Mount(engine.Group(prefix), svc)
}
