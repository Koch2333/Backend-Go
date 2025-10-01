package aicweb

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
	ts  TurnstileVerifier
	fs  FormService
}

func NewHandler(svc Service, ts TurnstileVerifier, fs FormService) *Handler {
	return &Handler{svc: svc, ts: ts, fs: fs}
}

// 让 gin.Context 实现 RequestCtx（turnstile.go 用到）
type ginCtx struct{ *gin.Context }

func (g ginCtx) ShouldBindBodyWithJSON(v any) error { return g.Context.ShouldBindJSON(v) }

// ---- 注册 ----
func (h *Handler) Register(c *gin.Context) {
	// Turnstile（若启用）
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
			c.JSON(http.StatusOK, NewFail(err, nil)) // 业务失败：HTTP 200
		default:
			c.JSON(http.StatusInternalServerError, NewFail(ErrInternalServerError, nil))
		}
		return
	}
	c.JSON(http.StatusOK, NewOK(nil))
}

// ---- 登录 ----
func (h *Handler) Login(c *gin.Context) {
	// Turnstile（若启用）
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
// POST /user/form
func (h *Handler) SubmitForm(c *gin.Context) {
	u := c.MustGet(ctxUserKey).(*user)

	// 读取并净化原始 JSON（递归剔除 password/token 等敏感键），1MB 上限
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
// GET /user/form?limit=50
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

	// 直接用 json.RawMessage 保持原样 JSON，不会被转义
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
