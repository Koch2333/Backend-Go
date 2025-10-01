package redirect

import (
	"backend-go/internal/redirect/envinit"
	"github.com/gin-gonic/gin"
)

func Mount(r *gin.RouterGroup) {
	svc, err := NewServiceFromEnv()
	if err != nil {
		panic("redirect service init failed: " + err.Error())
	}
	h := NewHandler(svc)
	r.GET("/:name", h.RedirectByName)
	r.GET("/pncs/:hwid", h.RedirectNFC)
}

func Attach(engine *gin.Engine) {
	envinit.Init() // redirect 专属 envinit
	grp := engine.Group("/api/redirect")
	Mount(grp)
}
