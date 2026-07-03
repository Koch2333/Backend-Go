package roundnfc

import (
	"crypto/subtle"
	"net/http"

	"backend-go/internal/auth"

	"github.com/gin-gonic/gin"
)

func adminRequired(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if validAppToken(c.GetHeader("X-App-Token"), svc.cfg.AdminAppToken) {
			c.Set(auth.ContextKeySubject, "app-token")
			c.Next()
			return
		}

		raw := auth.ExtractBearer(c.GetHeader("Authorization"))
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1, "message": "missing token"})
			return
		}
		claims, err := auth.ParseToken(svc.cfg.JWTSecret, raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1, "message": "invalid token"})
			return
		}
		c.Set(auth.ContextKeySubject, claims.Subject)
		c.Next()
	}
}

func validAppToken(got, want string) bool {
	if len(want) < 16 || got == "" || len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
