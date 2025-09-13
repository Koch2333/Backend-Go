package http

import (
	"backend-go/internal/integrations/aicweb"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, cfg *config.Config, deps Deps) {
	// ...主业务

	if cfg.IntegrationAICWebEnabled {
		grp := r.Group("/api/integrations/aicweb/v1",
			middleware.MetricsTag("aicweb"),
			middleware.RateLimit("aicweb"),
			// 如与主鉴权不同，可在此放置独立鉴权中间件
		)
		aicweb.Mount(grp, cfg, deps)
	}
}
