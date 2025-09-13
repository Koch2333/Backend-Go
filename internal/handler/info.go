package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// InfoHandler 结构体用于持有应用信息
type InfoHandler struct {
	CodeName string
	Version  string
	Commit   string
	Build    string
}

// NewInfoHandler 是 InfoHandler 的构造函数
func NewInfoHandler(version, commit, build string) *InfoHandler {
	return &InfoHandler{
		CodeName: "Roast", // 你可以硬编码，或者也作为参数传入
		Version:  version,
		Commit:   commit,
		Build:    build,
	}
}

// HandleStatus 处理 /status 请求
func (h *InfoHandler) HandleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}

// HandleVersion 处理 /version 请求
func (h *InfoHandler) HandleVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"codeName": h.CodeName,
		"version":  h.Version,
		"commit":   h.Commit,
		"build":    h.Build,
	})
}
