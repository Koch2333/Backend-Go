package roundnfc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"log"
	"net/http"
	"strings"

	"backend-go/internal/auth"

	"github.com/gin-gonic/gin"
)

const appTokenHeader = "X-RoundNFC-App-Token"

func newAppTokenPlain() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "rnfca_" + base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func appTokenHash(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func appTokenPrefix(token string) string {
	token = strings.TrimSpace(token)
	if len(token) <= 12 {
		return token
	}
	return token[:12]
}

func verifyStoredAppToken(ctx context.Context, svc *Service, token string) (bool, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return false, nil
	}
	return svc.store.VerifyAppToken(ctx, appTokenHash(token))
}

func appTokenRequired(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader(appTokenHeader))
		if raw == "" {
			raw = strings.TrimSpace(c.GetHeader("X-App-Token"))
		}
		ok, err := verifyStoredAppToken(c.Request.Context(), svc, raw)
		if err != nil {
			log.Printf("[roundnfc/app-auth] token check failed ip=%s err=%v", c.ClientIP(), err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "token check failed"})
			return
		}
		if !ok && validAppToken(raw, svc.cfg.AdminAppToken) {
			ok = true
		}
		if !ok {
			log.Printf("[roundnfc/app-auth] invalid token ip=%s ua=%q", c.ClientIP(), c.GetHeader("User-Agent"))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1, "message": "invalid app token"})
			return
		}
		c.Set(auth.ContextKeySubject, "app-token")
		c.Next()
	}
}
