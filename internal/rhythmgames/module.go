package rhythmgames

import (
	"backend-go/internal/bootstrap/plug"
	"github.com/gin-gonic/gin"
)

type modRG struct{}

func (modRG) Name() string          { return "rhythmgames" }
func (modRG) DefaultPrefix() string { return "/api/rhythmproper" }
func (modRG) DefaultEnabled() bool  { return true }
func (modRG) InitEnv()              {}

func (modRG) Mount(e *gin.Engine, p string) error {
	AttachTo(e, p)
	return nil
}

func init() { plug.Register(modRG{}) }
