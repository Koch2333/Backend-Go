package roundnfc

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type appHandler struct {
	svc       *Service
	apiPrefix string
}

func newAppHandler(svc *Service, apiPrefix string) *appHandler {
	return &appHandler{svc: svc, apiPrefix: strings.TrimRight(apiPrefix, "/")}
}

func (h *appHandler) ListStyleTemplates(c *gin.Context) {
	items, err := h.svc.store.ListBadgeStyleTemplates(c.Request.Context(), true)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range items {
		items[i] = h.svc.PublicStyleTemplate(c.Request.Context(), items[i], h.apiPrefix)
	}
	respondData(c, gin.H{"items": items})
}

type appBadgePayload struct {
	ID       string `json:"id"`
	StyleKey string `json:"styleKey"`
}

func (h *appHandler) UpsertBadgeStyle(c *gin.Context) {
	var p appBadgePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	id := strings.TrimSpace(p.ID)
	styleKey := strings.TrimSpace(p.StyleKey)
	if id == "" {
		respondError(c, http.StatusBadRequest, "id required")
		return
	}
	ok, err := h.svc.store.ValidBadgeStyleKey(c.Request.Context(), styleKey)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		respondError(c, http.StatusBadRequest, "invalid styleKey")
		return
	}

	b := &Badge{
		ID:       id,
		Title:    id,
		Type:     "badge",
		StyleKey: styleKey,
	}
	if cur, err := h.svc.store.GetBadge(c.Request.Context(), id); err == nil {
		b = cur
		b.StyleKey = styleKey
		if strings.TrimSpace(b.Title) == "" {
			b.Title = id
		}
		if strings.TrimSpace(b.Type) == "" {
			b.Type = "badge"
		}
	}
	if err := h.svc.store.UpsertBadge(c.Request.Context(), b); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[roundnfc/app] upsert badge style badgeId=%q styleKey=%q ip=%s", b.ID, b.StyleKey, c.ClientIP())
	respondData(c, b)
}

type coserPhotoPresignPayload struct {
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
}

func (h *appHandler) PresignCoserPhoto(c *gin.Context) {
	badgeID := strings.TrimSpace(c.Param("id"))
	if badgeID == "" {
		respondError(c, http.StatusBadRequest, "badgeId required")
		return
	}
	var p coserPhotoPresignPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	out, err := h.svc.PresignUpload(c.Request.Context(), badgeID, p.FileName, p.ContentType, "coser-photo")
	if err != nil {
		log.Printf("[roundnfc/app] presign coser photo failed badgeId=%q fileName=%q contentType=%q ip=%s err=%v",
			badgeID, p.FileName, p.ContentType, c.ClientIP(), err)
		if errors.Is(err, ErrCOSNotConfigured) {
			respondError(c, http.StatusServiceUnavailable, "cos not configured")
			return
		}
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	log.Printf("[roundnfc/app] presign coser photo badgeId=%q objectKey=%q contentType=%q expiresIn=%d ip=%s",
		badgeID, out.ObjectKey, p.ContentType, out.ExpiresIn, c.ClientIP())
	respondData(c, out)
}

type coserBindingPayload struct {
	CN             string `json:"cn"`
	PhotoObjectKey string `json:"photoObjectKey"`
	DeviceID       string `json:"deviceId"`
	TagUID         string `json:"tagUid"`
	WrittenAt      string `json:"writtenAt"`
}

func (h *appHandler) UpsertCoserBinding(c *gin.Context) {
	badgeID := strings.TrimSpace(c.Param("id"))
	if badgeID == "" {
		respondError(c, http.StatusBadRequest, "badgeId required")
		return
	}
	var p coserBindingPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	cn := strings.TrimSpace(p.CN)
	if cn == "" {
		respondError(c, http.StatusBadRequest, "cn required")
		return
	}
	photoKey := strings.TrimSpace(p.PhotoObjectKey)
	if photoKey == "" {
		respondError(c, http.StatusBadRequest, "photoObjectKey required")
		return
	}
	if isAbsoluteURL(photoKey) {
		respondError(c, http.StatusBadRequest, "photoObjectKey must be an object key")
		return
	}
	writtenAt := time.Now().UTC()
	if strings.TrimSpace(p.WrittenAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(p.WrittenAt))
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid writtenAt")
			return
		}
		writtenAt = t.UTC()
	}
	b := &BadgeCoserBinding{
		BadgeID:        badgeID,
		CN:             cn,
		PhotoObjectKey: photoKey,
		DeviceID:       strings.TrimSpace(p.DeviceID),
		TagUID:         strings.TrimSpace(p.TagUID),
		WrittenAt:      writtenAt,
	}
	if err := h.svc.store.UpsertBadgeCoserBinding(c.Request.Context(), b); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[roundnfc/app] upsert coser binding badgeId=%q cn=%q photoObjectKey=%q deviceId=%q tagUid=%q ip=%s",
		b.BadgeID, b.CN, b.PhotoObjectKey, b.DeviceID, b.TagUID, c.ClientIP())
	respondData(c, b)
}

func (h *appHandler) GetCoserBinding(c *gin.Context) {
	b, err := h.svc.store.GetBadgeCoserBinding(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(c, http.StatusNotFound, "not found")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(c, b)
}
