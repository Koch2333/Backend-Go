package roundnfc

import (
	"backend-go/internal/auth"
	"backend-go/internal/authflow"
	"backend-go/internal/roundnfc/envinit"

	"github.com/gin-gonic/gin"
)

// AttachTo 在 prefix 下挂载 RoundNFC 全部路由（公开 + 后台）。
func AttachTo(engine *gin.Engine, prefix string) error {
	envinit.Init()
	svc, err := NewServiceFromEnv()
	if err != nil {
		return err
	}
	if prefix == "" {
		prefix = "/api/roundnfc"
	}

	pub := newPublicHandler(svc, prefix)
	adm := newAdminHandler(svc)
	flow := authflow.New(svc.AuthFlowConfig())

	g := engine.Group(prefix)

	// public
	g.GET("/badges/:id", pub.GetBadge)
	g.POST("/badges/:id/photo-requests", pub.CreatePhotoRequest)
	g.POST("/badges/:id/autograph-requests", pub.CreateAutographRequest)
	g.POST("/uploads", pub.UploadAttachment)
	g.GET("/objects/:token", pub.GetObject)

	// admin — flow handles /login, /me, /totp/*, /webauthn/*
	admin := g.Group("/admin")
	flow.Mount(admin)

	// badge + request management (require valid JWT)
	authed := admin.Group("", auth.Required(svc.cfg.JWTSecret))
	authed.GET("/badges", adm.ListBadges)
	authed.POST("/badges", adm.UpsertBadge)
	authed.GET("/badges/:id", adm.GetBadge)
	authed.PUT("/badges/:id", adm.UpsertBadge)
	authed.DELETE("/badges/:id", adm.DeleteBadge)
	authed.POST("/badges/:id/image", adm.UploadBadgeImage)
	authed.GET("/photo-requests", adm.ListPhotoRequests)
	authed.PATCH("/photo-requests/:id", adm.UpdatePhotoStatus)
	authed.GET("/autograph-requests", adm.ListAutographRequests)
	authed.PATCH("/autograph-requests/:id", adm.UpdateAutographStatus)

	return nil
}
