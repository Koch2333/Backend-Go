package main

import (
	"log"

	"backend-go/internal/bootstrap/mod"
	"backend-go/internal/handler"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	// 侧效导入模块（触发 init -> 注册）
	_ "backend-go/internal/integrations/aicweb"
	_ "backend-go/internal/redirect"
)

var (
	Version = "dev"
	Commit  = "none"
	Build   = "local"
)

func main() {
	router := gin.Default()

	cfg := cors.DefaultConfig()
	cfg.AllowOrigins = []string{
		"https://koch2333.cn",
		"http://localhost:5173",
		"http://127.0.0.1:3000",
	}
	cfg.AllowCredentials = true
	cfg.AddAllowHeaders("CF-Turnstile-Response")
	router.Use(cors.New(cfg))

	info := handler.NewInfoHandler(Version, Commit, Build)
	router.GET("/status", info.HandleStatus)
	router.GET("/version", info.HandleVersion)

	// ★ 自动挂载所有注册模块
	mod.MountAll(router)

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Backend-Go is running.", "version": Version, "commit": Commit, "build": Build})
	})

	port := ":8080"
	log.Printf("服务器启动于 %s", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
