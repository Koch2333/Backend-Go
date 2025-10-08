package msconsent

import "github.com/gin-gonic/gin"

// 默认前缀 /auth/ms
func Attach(engine *gin.Engine) {
	AttachTo(engine, "/auth/ms")
}
