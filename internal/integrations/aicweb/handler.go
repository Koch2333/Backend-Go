package aicweb

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// POST /user/register
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewFail(ErrBadRequest, nil))
		return
	}
	if req.Email == "" || req.Password == "" || req.Username == "" {
		c.JSON(http.StatusBadRequest, NewFail(ErrBadRequest, nil))
		return
	}
	if err := h.svc.Register(c, &req); err != nil {
		switch err {
		case ErrEmailAlreadyUse:
			c.JSON(http.StatusOK, NewFail(err, nil)) // 与 aicweb 行为保持一致：HTTP 200 + 业务码
		default:
			c.JSON(http.StatusInternalServerError, NewFail(ErrInternalServerError, nil))
		}
		return
	}
	c.JSON(http.StatusOK, NewOK(nil))
}

// POST /user/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewFail(ErrBadRequest, nil))
		return
	}
	tok, err := h.svc.Login(c, &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewFail(ErrUnauthorized, nil))
		return
	}
	c.JSON(http.StatusOK, NewOK(LoginResponseData{AccessToken: tok}))
}

// GET /user/profile （演示：受保护接口）
func (h *Handler) Profile(c *gin.Context) {
	u := c.MustGet(ctxUserKey).(*user)
	c.JSON(http.StatusOK, NewOK(map[string]any{
		"id":       u.ID,
		"username": u.Username,
		"email":    u.Email,
	}))
}
