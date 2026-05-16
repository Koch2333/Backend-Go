package aicweb

import (
	"backend-go/internal/integrations/msconsent"
	"os"

	em "backend-go/internal/email"
	emenv "backend-go/internal/email/envinit"
	"backend-go/internal/integrations/aicweb/envinit"

	"github.com/gin-gonic/gin"
)

// Mount 把所有路由挂到传入的 RouterGroup 上。
func Mount(r *gin.RouterGroup) {
	var svc Service
	if s, err := NewServiceSQLiteFromEnv(); err == nil {
		svc = s
	} else {
		svc = NewServiceMemory()
	}

	ts := NewTurnstileFromEnv()

	fs, err := NewFormServiceFromEnv()
	if err != nil {
		panic("failed to init sqlite form service: " + err.Error())
	}

	emenv.Init()
	sender := em.NewSenderFromEnv()
	notify := NewEmailActivationNotifierFromEnv(sender)

	h := NewHandler(svc, ts, fs, notify)

	// 公共路由
	r.POST("/user/register", h.Register)
	r.POST("/user/login", h.Login)
	r.GET("/user/activate", h.Activate)
	r.GET("/user/profiles", h.ListProfiles)
	r.GET("/user/profile/:username", h.GetPublicProfile)

	// 受保护路由
	prv := r.Group("", AuthRequired(svc))
	{
		prv.GET("/user/profile", h.Profile)
		prv.PUT("/user/me/profile", h.UpdateMyProfile)
		prv.POST("/user/form", h.SubmitForm)
		prv.GET("/user/form", h.ListMyForms)
	}
}

func Attach(engine *gin.Engine) {
	envinit.Init()
	base := os.Getenv("AICWEB_BASE_PREFIX")
	if base == "" {
		base = "/api/aicweb"
	}
	AttachTo(engine, base)
}

func AttachTo(engine *gin.Engine, prefix string) {
	envinit.Init()
	if prefix == "" {
		prefix = "/api/aicweb"
	}
	grp := engine.Group(prefix)
	Mount(grp)
	msconsent.Attach(engine)
}
