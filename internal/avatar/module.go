package avatar

import (
	"backend-go/internal/avatar/envinit"
	"backend-go/internal/bootstrap/plug"

	"github.com/gin-gonic/gin"
)

type modAvatar struct{}

func (modAvatar) Name() string                        { return "avatar" }
func (modAvatar) DefaultPrefix() string               { return "/api/avatar" }
func (modAvatar) DefaultEnabled() bool                { return true }
func (modAvatar) InitEnv()                            { envinit.Init() }
func (modAvatar) Mount(e *gin.Engine, p string) error { AttachTo(e, p); return nil }

func init() { plug.Register(modAvatar{}) }
