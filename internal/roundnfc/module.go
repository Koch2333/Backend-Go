package roundnfc

import (
	"backend-go/internal/bootstrap/plug"
	"backend-go/internal/roundnfc/envinit"

	"github.com/gin-gonic/gin"
)

type modRoundNFC struct{}

func (modRoundNFC) Name() string                        { return "roundnfc" }
func (modRoundNFC) DefaultPrefix() string               { return "/api/roundnfc" }
func (modRoundNFC) DefaultEnabled() bool                { return true }
func (modRoundNFC) InitEnv()                            { envinit.Init() }
func (modRoundNFC) Mount(e *gin.Engine, p string) error { return AttachTo(e, p) }

func init() { plug.Register(modRoundNFC{}) }
