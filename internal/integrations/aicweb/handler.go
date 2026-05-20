package aicweb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc    Service
	ts     TurnstileVerifier
	fs     FormService
	notify ActivationNotifier
	avt    MediaUploader // nil = avatar upload disabled
	bnr    MediaUploader // nil = banner upload disabled
}

func NewHandler(svc Service, ts TurnstileVerifier, fs FormService, notify ActivationNotifier, avt, bnr MediaUploader) *Handler {
	return &Handler{svc: svc, ts: ts, fs: fs, notify: notify, avt: avt, bnr: bnr}
}

// ---- Turnstile 适配 ----
type ginCtx struct{ *gin.Context }

func (g ginCtx) GetHeader(k string) string          { return g.Context.GetHeader(k) }
func (g ginCtx) ClientIP() string                   { return g.Context.ClientIP() }
func (g ginCtx) ShouldBindBodyWithJSON(v any) error { return g.Context.ShouldBindJSON(v) }

type activationCreator interface {
	CreateActivationToken(ctx context.Context, email string) (string, error)
}
type activationActivator interface {
	ActivateByToken(ctx context.Context, token string) error
}

// openUploadReader returns a reader for the uploaded file, supporting both
// multipart/form-data (field name: "file") and raw binary body.
func openUploadReader(c *gin.Context) (io.ReadCloser, error) {
	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		f, _, err := c.Request.FormFile("file")
		return f, err
	}
	return c.Request.Body, nil
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
			c.JSON(http.StatusUnauthorized, NewFail(ErrUnauthorized, map[string]any{"reason": "NOT_ACTIVATED"}))
			return
		}
		c.JSON(http.StatusUnauthorized, NewFail(ErrUnauthorized, nil))
		return
	}
	c.JSON(http.StatusOK, NewOK(LoginResponseData{AccessToken: tok}))
}

// ---- 当前用户个人信息（需登录）----
func (h *Handler) Profile(c *gin.Context) {
	u := c.MustGet(ctxUserKey).(*user)
	c.JSON(http.StatusOK, NewOK(map[string]any{
		"id":       u.ID,
		"username": u.Username,
		"email":    u.Email,
	}))
}

// ---- 公开 profile 列表 ----
func (h *Handler) ListProfiles(c *gin.Context) {
	ps, ok := h.svc.(ProfileService)
	if !ok {
		c.JSON(http.StatusOK, NewOK([]PublicProfile{}))
		return
	}
	profiles, err := ps.ListPublicProfiles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewFail(ErrInternalServerError, nil))
		return
	}
	if profiles == nil {
		profiles = []PublicProfile{}
	}
	c.JSON(http.StatusOK, NewOK(profiles))
}

// ---- 按用户名获取公开 profile ----
func (h *Handler) GetPublicProfile(c *gin.Context) {
	username := c.Param("username")
	ps, ok := h.svc.(ProfileService)
	if !ok {
		c.JSON(http.StatusNotFound, NewFail(ErrNotFound, nil))
		return
	}
	profile, err := ps.GetPublicProfile(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewFail(ErrInternalServerError, nil))
		return
	}
	if profile == nil {
		c.JSON(http.StatusNotFound, NewFail(ErrNotFound, nil))
		return
	}
	c.JSON(http.StatusOK, NewOK(profile))
}

// ---- 更新当前用户 profile（需登录）----
func (h *Handler) UpdateMyProfile(c *gin.Context) {
	u := c.MustGet(ctxUserKey).(*user)
	ps, ok := h.svc.(ProfileService)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, NewFail(ErrInternalServerError, nil))
		return
	}
	var update ProfileUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, NewFail(ErrBadRequest, nil))
		return
	}
	// Preserve avatarUrl/bannerUrl — they are set only via dedicated upload endpoints.
	if existing, err := ps.GetPublicProfile(c.Request.Context(), u.Username); err == nil && existing != nil {
		update.AvatarUrl = existing.AvatarUrl
		update.BannerUrl = existing.BannerUrl
	}
	if err := ps.UpdateMyProfile(c.Request.Context(), u.ID, update); err != nil {
		c.JSON(http.StatusInternalServerError, NewFail(ErrInternalServerError, nil))
		return
	}
	profile, _ := ps.GetPublicProfile(c.Request.Context(), u.Username)
	c.JSON(http.StatusOK, NewOK(profile))
}

// ---- 上传头像（需登录）----
func (h *Handler) UploadAvatar(c *gin.Context) {
	if h.avt == nil {
		c.JSON(http.StatusNotImplemented, NewFail(ErrInternalServerError, nil))
		return
	}
	u := c.MustGet(ctxUserKey).(*user)
	ps, ok := h.svc.(ProfileService)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, NewFail(ErrInternalServerError, nil))
		return
	}
	r, err := openUploadReader(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, NewFail(ErrBadRequest, map[string]any{"reason": "file missing"}))
		return
	}
	defer r.Close()
	url, err := h.avt.Upload(r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing, _ := ps.GetPublicProfile(c.Request.Context(), u.Username)
	update := profileFromExisting(existing)
	update.AvatarUrl = url
	if err := ps.UpdateMyProfile(c.Request.Context(), u.ID, update); err != nil {
		c.JSON(http.StatusInternalServerError, NewFail(ErrInternalServerError, nil))
		return
	}
	c.JSON(http.StatusOK, NewOK(map[string]string{"url": url}))
}

// ---- 上传横幅（需登录）----
func (h *Handler) UploadBanner(c *gin.Context) {
	if h.bnr == nil {
		c.JSON(http.StatusNotImplemented, NewFail(ErrInternalServerError, nil))
		return
	}
	u := c.MustGet(ctxUserKey).(*user)
	ps, ok := h.svc.(ProfileService)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, NewFail(ErrInternalServerError, nil))
		return
	}
	r, err := openUploadReader(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, NewFail(ErrBadRequest, map[string]any{"reason": "file missing"}))
		return
	}
	defer r.Close()
	url, err := h.bnr.Upload(r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing, _ := ps.GetPublicProfile(c.Request.Context(), u.Username)
	update := profileFromExisting(existing)
	update.BannerUrl = url
	if err := ps.UpdateMyProfile(c.Request.Context(), u.ID, update); err != nil {
		c.JSON(http.StatusInternalServerError, NewFail(ErrInternalServerError, nil))
		return
	}
	c.JSON(http.StatusOK, NewOK(map[string]string{"url": url}))
}

// ---- 表单提交 ----
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

	// Sync messageToSchool / messageToUnderclassmen from form to public profile.
	if ps, ok := h.svc.(ProfileService); ok {
		var msgs struct {
			MessageToSchool        string `json:"messageToSchool"`
			MessageToUnderclassmen string `json:"messageToUnderclassmen"`
		}
		if json.Unmarshal(payload, &msgs) == nil {
			existing, _ := ps.GetPublicProfile(c.Request.Context(), u.Username)
			update := profileFromExisting(existing)
			if msgs.MessageToSchool != "" {
				update.MessageToSchool = msgs.MessageToSchool
			}
			if msgs.MessageToUnderclassmen != "" {
				update.MessageToUnderclassmen = msgs.MessageToUnderclassmen
			}
			_ = ps.UpdateMyProfile(c.Request.Context(), u.ID, update)
		}
	}

	c.JSON(http.StatusOK, NewOK(map[string]any{"ok": true}))
}

// ---- 表单查询 ----
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

// ---- 激活 ----
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
