package redirect

import (
	"backend-go/internal/bootstrap/plug"
	"backend-go/internal/redirect/envinit"
	"github.com/gin-gonic/gin"
)

type modRedirect struct{}

func (modRedirect) Name() string          { return "redirect" }
func (modRedirect) DefaultPrefix() string { return "/api/redirect" }
func (modRedirect) DefaultEnabled() bool  { return true }
func (modRedirect) InitEnv()              { envinit.Init() }
func (modRedirect) Mount(e *gin.Engine, p string) error {
	AttachTo(e, p)
	return nil
}

func init() { plug.Register(modRedirect{}) }
