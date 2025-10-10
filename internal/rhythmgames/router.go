package rhythmgames

import "github.com/gin-gonic/gin"

func Mount(r *gin.RouterGroup) {
	// DX Rating SVG
	r.GET("/:game/dxrating.svg", handleDXRating)
}

func AttachTo(engine *gin.Engine, prefix string) {
	if prefix == "" {
		prefix = "/api/rhythmproper"
	}
	Mount(engine.Group(prefix))
}
