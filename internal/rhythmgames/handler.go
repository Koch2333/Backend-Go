package rhythmgames

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	httpTimeout = 8 * time.Second
	cacheTTL    = 5 * time.Minute
	memCache    = NewTTLCache[*RatingResult](cacheTTL)
)

func handleDXRating(c *gin.Context) {
	game := strings.ToLower(strings.TrimSpace(c.Param("game")))
	user := strings.TrimSpace(c.Query("user"))

	write := func(svg string, cacheable bool) {
		c.Header("Content-Type", "image/svg+xml; charset=utf-8")
		if cacheable {
			c.Header("Cache-Control", "max-age=300, public")
		} else {
			c.Header("Cache-Control", "no-cache")
		}
		c.String(http.StatusOK, svg)
	}

	p, err := GetProvider(game)
	if err != nil || user == "" {
		write(RenderDXRatingSVG(0), false)
		return
	}

	key := hashKey("dx", game, user)
	if v, ok := memCache.Get(key); ok && v != nil {
		write(RenderDXRatingSVG(v.Rating), true)
		return
	}

	ctx, cancel := context.WithTimeout(c, httpTimeout+1*time.Second)
	defer cancel()
	res, err := p.FetchRating(ctx, UserID{Username: user})
	if err != nil || res == nil {
		write(RenderDXRatingSVG(0), true)
		return
	}

	memCache.Set(key, res)
	write(RenderDXRatingSVG(res.Rating), true)
}

func hashKey(parts ...string) string {
	h := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:])
}
