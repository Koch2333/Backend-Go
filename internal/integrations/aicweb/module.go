package aicweb

import (
	"backend-go/internal/bootstrap/plug"
	"backend-go/internal/integrations/aicweb/envinit"

	"github.com/gin-gonic/gin"
)

type modAICWeb struct{}

func (modAICWeb) Name() string                        { return "aicweb" }
func (modAICWeb) DefaultPrefix() string               { return "/api/aicweb" }
func (modAICWeb) DefaultEnabled() bool                { return true }
func (modAICWeb) InitEnv()                            { envinit.Init() }
func (modAICWeb) Mount(e *gin.Engine, p string) error { AttachTo(e, p); return nil }

func init() { plug.Register(modAICWeb{}) }
