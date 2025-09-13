// cmd/web/main.go
package main

import (
	"log"

	"backend-go/internal/handler"
	integ "backend-go/internal/integrations/aicweb" // AICWeb 开发版独立 API
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// 这些变量保留在 main 包，因为它们是在编译时被注入的
var (
	Version = "dev"
	Commit  = "none"
	Build   = "local"
)

func main() {
	// 1) 初始化 Gin
	router := gin.Default()

	// 2) CORS（开发环境常见前端来源：你的域名 + 本地端口）
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"https://koch2333.cn",
		"http://localhost:3000",
		"http://127.0.0.1:3000",
	}
	config.AllowCredentials = true
	router.Use(cors.New(config))

	// 3) 信息接口（保留你现有的 InfoHandler）
	infoHandler := handler.NewInfoHandler(Version, Commit, Build)
	router.GET("/status", infoHandler.HandleStatus)
	router.GET("/version", infoHandler.HandleVersion)

	// 4) AICWeb 开发版独立 API（不依赖 aicweb 源码，默认挂到 /api/aicweb）
	integ.Attach(router)

	// 5) 根路由（简易欢迎页，避免 404；不再做重定向）
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Backend-Go is running.",
			"version": Version,
			"commit":  Commit,
			"build":   Build,
		})
	})

	// 6) 启动
	port := ":8080"
	log.Printf("服务器启动于 %s", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
