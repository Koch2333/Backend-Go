package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.GET("/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "OK",
		})
	})
	fmt.Printf("Backend started at 0.0.0.0:8080")
	router.Run() // 监听并在 0.0.0.0:8080 上启动服务

}
