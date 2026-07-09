package roundnfc

import (
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
	adm := newAdminHandler(svc, prefix)
	apph := newAppHandler(svc, prefix)
	flow := authflow.New(svc.AuthFlowConfig())

	g := engine.Group(prefix)

	// public
	g.GET("/badges/:id", pub.GetBadge)
	g.POST("/badges/:id/photo-requests", pub.CreatePhotoRequest)
	g.POST("/badges/:id/autograph-requests", pub.CreateAutographRequest)
	g.POST("/uploads", pub.UploadAttachment)
	g.GET("/objects/:token", pub.GetObject)
	g.GET("/cos-objects/:token", pub.RedirectCOSObject)

	// admin — flow handles /login, /me, /totp/*, /webauthn/*
	admin := g.Group("/admin")
	flow.Mount(admin)

	// badge + request management (require valid JWT or app token)
	authed := admin.Group("", adminRequired(svc))
	authed.GET("/badges", adm.ListBadges)
	authed.POST("/badges", adm.UpsertBadge)
	authed.GET("/badges/:id", adm.GetBadge)
	authed.PUT("/badges/:id", adm.UpsertBadge)
	authed.DELETE("/badges/:id", adm.DeleteBadge)
	authed.POST("/badges/:id/image", adm.UploadBadgeImage)
	authed.GET("/styles", apph.ListStyleTemplates)
	authed.GET("/style-templates", adm.ListStyleTemplates)
	authed.POST("/style-templates", adm.UpsertStyleTemplate)
	authed.PUT("/style-templates/:key", adm.UpsertStyleTemplate)
	authed.POST("/style-templates/:key/image", adm.UploadStyleTemplateImage)
	authed.DELETE("/style-templates/:key", adm.DeleteStyleTemplate)
	authed.POST("/uploads/presign", adm.PresignUpload)
	authed.POST("/nfc-writes", adm.CreateNFCWrite)
	authed.GET("/photo-requests", adm.ListPhotoRequests)
	authed.PATCH("/photo-requests/:id", adm.UpdatePhotoStatus)
	authed.GET("/autograph-requests", adm.ListAutographRequests)
	authed.PATCH("/autograph-requests/:id", adm.UpdateAutographStatus)
	authed.GET("/app-tokens", adm.ListAppTokens)
	authed.POST("/app-tokens", adm.CreateAppToken)
	authed.PATCH("/app-tokens/:id", adm.UpdateAppToken)
	authed.DELETE("/app-tokens/:id", adm.DeleteAppToken)

	// Android writer app. Pair by scanning the admin-generated QR code, then
	// authenticate with X-RoundNFC-App-Token.
	app := g.Group("/app", appTokenRequired(svc))
	app.GET("/styles", apph.ListStyleTemplates)
	app.GET("/style-templates", apph.ListStyleTemplates)
	app.GET("/badges", adm.ListBadges)
	app.GET("/badges/:id", adm.GetBadge)
	app.POST("/badges", apph.UpsertBadgeStyle)
	app.POST("/badges/:id/coser-photo/presign", apph.PresignCoserPhoto)
	app.GET("/badges/:id/coser-binding", apph.GetCoserBinding)
	app.POST("/badges/:id/coser-binding", apph.UpsertCoserBinding)
	app.POST("/uploads/presign", adm.PresignUpload)
	app.POST("/nfc-writes", adm.CreateNFCWrite)

	return nil
}
