package aicweb

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
	ts  TurnstileVerifier
}

func NewHandler(svc Service, ts TurnstileVerifier) *Handler { return &Handler{svc: svc, ts: ts} }

// 让 gin.Context 实现 RequestCtx
type ginCtx struct{ *gin.Context }

func (g ginCtx) ShouldBindBodyWithJSON(v any) error { return g.Context.ShouldBindJSON(v) } // 用一次性绑定

// POST /user/register
func (h *Handler) Register(c *gin.Context) {
	// Turnstile 校验（若启用）
	if h.ts != nil && h.ts.Enabled() {
		token, err := getTurnstileToken(ginCtx{c})
		if err != nil {
			c.JSON(http.StatusBadRequest, NewFail(ErrBadRequest, map[string]any{"reason": "missing turnstile token"}))
			return
		}
		if ok, codes, err := h.ts.Verify(c, token, c.ClientIP()); err != nil || !ok {
			c.JSON(http.StatusBadRequest, NewFail(ErrBadRequest, map[string]any{"turnstile": codes}))
			return
		}
	}

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
			c.JSON(http.StatusOK, NewFail(err, nil)) // 业务失败：HTTP 200 + code
		default:
			c.JSON(http.StatusInternalServerError, NewFail(ErrInternalServerError, nil))
		}
		return
	}
	c.JSON(http.StatusOK, NewOK(nil))
}

// POST /user/login
func (h *Handler) Login(c *gin.Context) {
	// Turnstile 校验（若启用）
	if h.ts != nil && h.ts.Enabled() {
		token, err := getTurnstileToken(ginCtx{c})
		if err != nil {
			c.JSON(http.StatusBadRequest, NewFail(ErrBadRequest, map[string]any{"reason": "missing turnstile token"}))
			return
		}
		if ok, codes, err := h.ts.Verify(c, token, c.ClientIP()); err != nil || !ok {
			c.JSON(http.StatusBadRequest, NewFail(ErrBadRequest, map[string]any{"turnstile": codes}))
			return
		}
	}

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

// GET /user/profile（原样）
func (h *Handler) Profile(c *gin.Context) {
	u := c.MustGet(ctxUserKey).(*user)
	c.JSON(http.StatusOK, NewOK(map[string]any{
		"id":       u.ID,
		"username": u.Username,
		"email":    u.Email,
	}))
}
