package roundnfc

import (
	"net/http"
	"strings"

	"backend-go/internal/risk"
	"backend-go/pkg/objstore"

	"github.com/gin-gonic/gin"
)

type publicHandler struct {
	svc       *Service
	apiPrefix string
}

func newPublicHandler(svc *Service, apiPrefix string) *publicHandler {
	return &publicHandler{svc: svc, apiPrefix: apiPrefix}
}

func (h *publicHandler) GetBadge(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		respondError(c, http.StatusBadRequest, "id required")
		return
	}
	b, err := h.svc.store.GetBadge(c.Request.Context(), id)
	if err != nil {
		if err == ErrNotFound {
			respondError(c, http.StatusNotFound, "not found")
			return
		}
		respondError(c, http.StatusInternalServerError, "internal error")
		return
	}
	respondData(c, h.svc.PublicBadge(c.Request.Context(), b, h.apiPrefix))
}

func (h *publicHandler) ListSocialLinks(c *gin.Context) {
	items, err := h.svc.store.ListSocialLinks(c.Request.Context(), true)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal error")
		return
	}
	respondData(c, gin.H{"items": items})
}

type photoRequestPayload struct {
	BadgeID        string   `json:"badgeId"`
	Name           string   `json:"name"`
	Contact        string   `json:"contact"`
	Message        string   `json:"message"`
	AttachmentKeys []string `json:"attachmentKeys"`
	TurnstileToken string   `json:"turnstileToken"`
}

func (h *publicHandler) CreatePhotoRequest(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var p photoRequestPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	if id == "" || p.Name == "" || p.Contact == "" {
		respondError(c, http.StatusBadRequest, "name and contact required")
		return
	}
	if !h.svc.rl.Allow("photo:" + c.ClientIP() + ":" + id) {
		respondError(c, http.StatusTooManyRequests, "too many requests")
		return
	}
	if !h.verifyTurnstile(c, p.TurnstileToken) {
		return
	}
	req := &PhotoRequest{
		ID:             newID("ph"),
		BadgeID:        id,
		Name:           strings.TrimSpace(p.Name),
		Contact:        strings.TrimSpace(p.Contact),
		Message:        strings.TrimSpace(p.Message),
		AttachmentKeys: p.AttachmentKeys,
		IPHash:         hashIP(c.ClientIP(), h.svc.cfg.AdminUsername),
	}
	if err := h.svc.store.InsertPhotoRequest(c.Request.Context(), req); err != nil {
		respondError(c, http.StatusInternalServerError, "internal error")
		return
	}
	respondData(c, gin.H{"requestId": req.ID})
}

type autographRequestPayload struct {
	BadgeID        string   `json:"badgeId"`
	Name           string   `json:"name"`
	Contact        string   `json:"contact"`
	Target         string   `json:"target"`
	Content        string   `json:"content"`
	AttachmentKeys []string `json:"attachmentKeys"`
	TurnstileToken string   `json:"turnstileToken"`
}

func (h *publicHandler) CreateAutographRequest(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var p autographRequestPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	if id == "" || p.Name == "" || p.Contact == "" || p.Content == "" {
		respondError(c, http.StatusBadRequest, "name, contact and content required")
		return
	}
	if !h.svc.rl.Allow("auto:" + c.ClientIP() + ":" + id) {
		respondError(c, http.StatusTooManyRequests, "too many requests")
		return
	}
	if !h.verifyTurnstile(c, p.TurnstileToken) {
		return
	}
	req := &AutographRequest{
		ID:             newID("au"),
		BadgeID:        id,
		Name:           strings.TrimSpace(p.Name),
		Contact:        strings.TrimSpace(p.Contact),
		Target:         strings.TrimSpace(p.Target),
		Content:        strings.TrimSpace(p.Content),
		AttachmentKeys: p.AttachmentKeys,
		IPHash:         hashIP(c.ClientIP(), h.svc.cfg.AdminUsername),
	}
	if err := h.svc.store.InsertAutographRequest(c.Request.Context(), req); err != nil {
		respondError(c, http.StatusInternalServerError, "internal error")
		return
	}
	respondData(c, gin.H{"requestId": req.ID})
}

func (h *publicHandler) UploadAttachment(c *gin.Context) {
	if !h.svc.rl.Allow("upload:" + c.ClientIP()) {
		respondError(c, http.StatusTooManyRequests, "too many requests")
		return
	}
	if !h.verifyTurnstile(c, c.PostForm("turnstileToken")) {
		return
	}
	f, _, err := c.Request.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "file missing")
		return
	}
	defer f.Close()
	key, mime, size, err := h.svc.IngestImage(c.Request.Context(), "uploads", f)
	if err != nil {
		switch err {
		case ErrTooLarge:
			respondError(c, http.StatusRequestEntityTooLarge, "file too large")
		case ErrUnsupportedMedia:
			respondError(c, http.StatusUnsupportedMediaType, "unsupported media")
		default:
			respondError(c, http.StatusInternalServerError, "internal error")
		}
		return
	}
	respondData(c, gin.H{"key": key, "mime": mime, "size": size})
}

// GET /objects/:token 返回 blob（一次性消费）。前端 fetch 后 createObjectURL 即可。
func (h *publicHandler) GetObject(c *gin.Context) {
	rc, meta, err := h.svc.ResolveObject(c.Request.Context(), c.Param("token"))
	if err != nil {
		switch err {
		case objstore.ErrTokenExpired, objstore.ErrTokenConsumed:
			c.String(http.StatusGone, "link expired")
		case objstore.ErrTokenInvalid:
			c.String(http.StatusForbidden, "forbidden")
		case objstore.ErrNotFound:
			c.String(http.StatusNotFound, "not found")
		default:
			c.String(http.StatusInternalServerError, "internal error")
		}
		return
	}
	defer rc.Close()
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.DataFromReader(http.StatusOK, meta.Size, meta.ContentType, rc, nil)
}

// GET /cos-objects/:token 消费一次性 token，并 302 到短时 COS 签名 GET URL。
func (h *publicHandler) RedirectCOSObject(c *gin.Context) {
	u, err := h.svc.ResolveCOSObjectURL(c.Request.Context(), c.Param("token"))
	if err != nil {
		switch err {
		case objstore.ErrTokenExpired, objstore.ErrTokenConsumed:
			c.String(http.StatusGone, "link expired")
		case objstore.ErrTokenInvalid:
			c.String(http.StatusForbidden, "forbidden")
		case ErrCOSNotConfigured:
			c.String(http.StatusServiceUnavailable, "cos not configured")
		default:
			c.String(http.StatusInternalServerError, "internal error")
		}
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, u)
}

func (h *publicHandler) verifyTurnstile(c *gin.Context, jsonToken string) bool {
	tok := jsonToken
	if tok == "" {
		tok = c.GetHeader("CF-Turnstile-Response")
	}
	ok, err := risk.VerifyTurnstile(c.Request.Context(), h.svc.cfg.TurnstileSecret, tok, c.ClientIP())
	if err != nil {
		respondError(c, http.StatusBadGateway, "turnstile verify failed")
		return false
	}
	if !ok {
		respondError(c, http.StatusForbidden, "turnstile verification failed")
		return false
	}
	return true
}
