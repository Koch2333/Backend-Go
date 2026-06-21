package comments

import (
	"backend-go/internal/auth"
	"backend-go/internal/comments/envinit"

	"github.com/gin-gonic/gin"
)

func AttachTo(engine *gin.Engine, prefix string) error {
	envinit.Init()
	svc, err := NewServiceFromEnv()
	if err != nil {
		return err
	}
	if prefix == "" {
		prefix = "/api/comments"
	}

	pub := newPublicHandler(svc)
	adm := newAdminHandler(svc)

	g := engine.Group(prefix)

	g.GET("/comments", pub.ListComments)
	g.POST("/comments", pub.CreateComment)

	admin := g.Group("/admin", auth.Required(svc.cfg.JWTSecret))
	admin.GET("/comments", adm.ListAll)
	admin.PATCH("/comments/:id", adm.UpdateStatus)
	admin.DELETE("/comments/:id", adm.Delete)

	return nil
}
