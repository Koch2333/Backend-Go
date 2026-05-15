// Package auth 提供后台账号鉴权所需的最小工具集：JWT + bcrypt + gin 中间件。
package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const ContextKeySubject = "auth.subject"

type Claims struct {
	Subject string `json:"sub"`
	jwt.RegisteredClaims
}

func IssueToken(secret []byte, subject string, ttl time.Duration) (string, time.Time, error) {
	exp := time.Now().Add(ttl)
	claims := Claims{
		Subject: subject,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(secret)
	return s, exp, err
}

func ParseToken(secret []byte, raw string) (*Claims, error) {
	c := &Claims{}
	_, err := jwt.ParseWithClaims(raw, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func ExtractBearer(h string) string {
	const p = "Bearer "
	if strings.HasPrefix(h, p) {
		return strings.TrimSpace(h[len(p):])
	}
	return ""
}

// Required 不通过则 401。
func Required(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := ExtractBearer(c.GetHeader("Authorization"))
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1, "message": "missing token"})
			return
		}
		claims, err := ParseToken(secret, raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1, "message": "invalid token"})
			return
		}
		c.Set(ContextKeySubject, claims.Subject)
		c.Next()
	}
}
