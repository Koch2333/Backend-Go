package maimai

import (
	"backend-go/internal/bootstrap/plug"
	"github.com/gin-gonic/gin"
)

type modMaimai struct{}

func (modMaimai) Name() string                        { return "rhythmgames.maimai" }
func (modMaimai) DefaultPrefix() string               { return "" }
func (modMaimai) DefaultEnabled() bool                { return true }
func (modMaimai) InitEnv()                            {}
func (modMaimai) Mount(e *gin.Engine, p string) error { return nil }

func init() { plug.Register(modMaimai{}) }
