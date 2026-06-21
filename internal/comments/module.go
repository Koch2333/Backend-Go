package comments

import (
	"backend-go/internal/bootstrap/plug"
	"backend-go/internal/comments/envinit"

	"github.com/gin-gonic/gin"
)

type modComments struct{}

func (modComments) Name() string                        { return "comments" }
func (modComments) DefaultPrefix() string               { return "/api/comments" }
func (modComments) DefaultEnabled() bool                { return true }
func (modComments) InitEnv()                            { envinit.Init() }
func (modComments) Mount(e *gin.Engine, p string) error { return AttachTo(e, p) }

func init() { plug.Register(modComments{}) }
