package main

import (
	"github.com/gin-gonic/gin"
)

var (
	Version = "dev"
	Commit  = "none"  //Current commit
	Build   = "local" //Building time
)

func main() {
	router := gin.Default()
	router.GET("/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "OK",
		})
	})
	router.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"codeName": "Haru",
			"version":  Version,
			"commit":   Commit,
			"build":    Build,
		})
	})
	router.Run() // 监听并在 0.0.0.0:8080 上启动服务

}
