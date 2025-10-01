package redirect

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

// GET /api/redirect/:name
func (h *Handler) RedirectByName(c *gin.Context) {
	name := c.Param("name")
	url, hit, err := h.svc.ResolveByName(name)
	if err != nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}
	if hit {
		c.Header("Cache-Control", "no-store")
		c.Redirect(http.StatusFound, url)
		return
	}
	// 未命中走兜底
	if url != "" {
		c.Header("Cache-Control", "no-store")
		c.Redirect(http.StatusFound, url)
		return
	}
	c.String(http.StatusNotFound, "not found")
}

// GET /api/redirect/pncs/:hwid
func (h *Handler) RedirectNFC(c *gin.Context) {
	hwid := c.Param("hwid")
	url, err := h.svc.ResolveNFC(hwid)
	if err != nil || url == "" {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, url)
}
