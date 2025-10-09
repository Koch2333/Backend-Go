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

	memCache = NewTTLCache[*RatingResult](cacheTTL)
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
		// 未知游戏/无 user：给 0 值徽章（仍优先上游）
		if svg := RenderUsagiDXBadgeExactUpstream(0); svg != "" {
			write(svg, false)
			return
		}
		write(RenderUsagiDXBadge(0), false)
		return
	}

	key := hashKey("dx", game, user)
	if v, ok := memCache.Get(key); ok && v != nil {
		if svg := RenderUsagiDXBadgeExactUpstream(v.Rating); svg != "" {
			write(svg, true)
			return
		}
		write(RenderUsagiDXBadge(v.Rating), true)
		return
	}

	ctx, cancel := context.WithTimeout(c, httpTimeout+1*time.Second)
	defer cancel()
	res, err := p.FetchRating(ctx, UserID{Username: user})
	if err != nil || res == nil {
		if svg := RenderUsagiDXBadgeExactUpstream(0); svg != "" {
			write(svg, true)
			return
		}
		write(RenderUsagiDXBadge(0), true)
		return
	}

	memCache.Set(key, res)
	if svg := RenderUsagiDXBadgeExactUpstream(res.Rating); svg != "" {
		write(svg, true)
		return
	}
	write(RenderUsagiDXBadge(res.Rating), true)
}

func hashKey(parts ...string) string {
	h := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:])
}
