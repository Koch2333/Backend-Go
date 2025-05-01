package main

import (
	"github.com/gin-contrib/cors"
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
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"https://koch2333.cn"}
	// config.AllowOrigins = []string{"http://google.com", "http://facebook.com"}
	// config.AllowAllOrigins = true

	router.Use(cors.New(config))
	router.Run() // 监听并在 0.0.0.0:8080 上启动服务

}
