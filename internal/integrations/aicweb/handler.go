package aicweb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc    Service
	ts     TurnstileVerifier
	fs     FormService
	notify ActivationNotifier
}

func NewHandler(svc Service, ts TurnstileVerifier, fs FormService, notify ActivationNotifier) *Handler {
	return &Handler{svc: svc, ts: ts, fs: fs, notify: notify}
}

// ---- Turnstile 适配：让 ginCtx 实现 RequestCtx ----
type ginCtx struct{ *gin.Context }

func (g ginCtx) GetHeader(k string) string          { return g.Context.GetHeader(k) }
func (g ginCtx) ClientIP() string                   { return g.Context.ClientIP() }
func (g ginCtx) ShouldBindBodyWithJSON(v any) error { return g.Context.ShouldBindJSON(v) }

// ---- 激活用的小接口（基于 context.Context）----
type activationCreator interface {
	CreateActivationToken(ctx context.Context, email string) (string, error)
}
type activationActivator interface {
	ActivateByToken(ctx context.Context, token string) error
}

// ---- 注册 ----
func (h *Handler) Register(c *gin.Context) {
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

	if err := h.svc.Register(c.Request.Context(), &req); err != nil {
		switch err {
		case ErrEmailAlreadyUse:
			c.JSON(http.StatusOK, NewFail(err, nil))
		default:
			c.JSON(http.StatusInternalServerError, NewFail(ErrInternalServerError, nil))
		}
		return
	}

	if st, ok := h.svc.(activationCreator); ok && h.notify != nil {
		if tok, err := st.CreateActivationToken(c.Request.Context(), req.Email); err == nil {
			_ = h.notify.SendActivation(req.Email, tok)
		}
	}

	c.JSON(http.StatusOK, NewOK(map[string]any{"registered": true}))
}

// ---- 登录 ----
func (h *Handler) Login(c *gin.Context) {
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
	tok, err := h.svc.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrNotActivated) {
			// ★ 前端可据此提示“账号未激活”
			c.JSON(http.StatusUnauthorized, NewFail(ErrUnauthorized, map[string]any{"reason": "NOT_ACTIVATED"}))
			return
		}
		c.JSON(http.StatusUnauthorized, NewFail(ErrUnauthorized, nil))
		return
	}
	c.JSON(http.StatusOK, NewOK(LoginResponseData{AccessToken: tok}))
}

// ---- 个人信息 ----
func (h *Handler) Profile(c *gin.Context) {
	u := c.MustGet(ctxUserKey).(*user)
	c.JSON(http.StatusOK, NewOK(map[string]any{
		"id":       u.ID,
		"username": u.Username,
		"email":    u.Email,
	}))
}

// ---- 表单提交（受保护）----
func (h *Handler) SubmitForm(c *gin.Context) {
	u := c.MustGet(ctxUserKey).(*user)
	payload, err := SanitizeRawJSON(c.Request.Body, 1<<20)
	if err != nil || len(payload) == 0 {
		c.JSON(http.StatusBadRequest, NewFail(ErrBadRequest, map[string]any{"reason": "invalid json"}))
		return
	}
	if err := h.fs.Submit(u.ID, c.ClientIP(), c.GetHeader("User-Agent"), payload); err != nil {
		c.JSON(http.StatusInternalServerError, NewFail(ErrInternalServerError, nil))
		return
	}
	c.JSON(http.StatusOK, NewOK(map[string]any{"ok": true}))
}

// ---- 表单查询（受保护）----
func (h *Handler) ListMyForms(c *gin.Context) {
	u := c.MustGet(ctxUserKey).(*user)
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	items, err := h.fs.List(u.ID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewFail(ErrInternalServerError, nil))
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, map[string]any{
			"id":         it.ID,
			"payload":    json.RawMessage(it.PayloadRaw),
			"created_at": it.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, NewOK(map[string]any{"list": out}))
}

// ---- 激活：GET /user/activate?token=... ----
func (h *Handler) Activate(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, NewFail(ErrBadRequest, nil))
		return
	}
	if st, ok := h.svc.(activationActivator); ok {
		if err := st.ActivateByToken(c.Request.Context(), token); err != nil {
			c.JSON(http.StatusUnauthorized, NewFail(ErrUnauthorized, nil))
			return
		}
		c.JSON(http.StatusOK, NewOK(map[string]any{"activated": true}))
		return
	}
	c.JSON(http.StatusNotFound, NewFail(ErrNotFound, nil))
}
