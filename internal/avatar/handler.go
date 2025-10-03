package avatar

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

// POST /api/avatar
// 支持 multipart/form-data（字段名：file）或原始二进制体）
// 仅返回 { "avatarId": "<md5hex>" }
func (h *Handler) Upload(c *gin.Context) {
	var reader io.Reader
	ct := c.GetHeader("Content-Type")

	if strings.HasPrefix(ct, "multipart/form-data") {
		f, _, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file missing"})
			return
		}
		defer safeClose(f)
		reader = f
	} else {
		reader = c.Request.Body
	}

	id, _, _, err := h.svc.ProcessAndStore(reader)
	if err != nil {
		if IsTooLarge(err) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"avatarId": id})
}

// GET /api/avatar/:id
// 只返回 { "avatarId": "<md5hex>" }；若不存在则 404
func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	fp := filepath.Join(h.svc.Dir, id+".webp")
	if _, err := os.Stat(fp); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"avatarId": id})
}

func safeClose(f multipart.File) { _ = f.Close() }
