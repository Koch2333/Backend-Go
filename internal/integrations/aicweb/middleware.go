package aicweb

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	ctxUserKey = "aicweb.user"
)

// AuthRequired 从 Header: Authorization: Bearer <token> 获取令牌并校验
func AuthRequired(svc Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if len(h) < 8 || h[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, NewFail(ErrUnauthorized, nil))
			return
		}
		tok := h[7:]
		u, err := svc.Validate(c, tok)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, NewFail(ErrUnauthorized, nil))
			return
		}
		c.Set(ctxUserKey, u)
		c.Next()
	}
}
