package roundnfc

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func respondData(c *gin.Context, data any) {
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": data})
}

func respondError(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"code": status, "message": msg, "data": nil})
}

func pageParams(c *gin.Context) (limit, offset int) {
	limit = 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}
