package roundnfc

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type adminHandler struct{ svc *Service }

func newAdminHandler(svc *Service) *adminHandler { return &adminHandler{svc: svc} }

type badgeUpsertPayload struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Series      string `json:"series"`
	Type        string `json:"type"`
	StyleKey    string `json:"styleKey"`
	ImageURL    string `json:"imageUrl"`
	Description string `json:"description"`
	SerialNo    string `json:"serialNo"`
	ReleasedAt  string `json:"releasedAt"`
}

func (h *adminHandler) ListBadges(c *gin.Context) {
	limit, offset := pageParams(c)
	items, total, err := h.svc.store.ListBadges(c.Request.Context(), c.Query("q"), limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(c, gin.H{"items": items, "total": total})
}

func (h *adminHandler) GetBadge(c *gin.Context) {
	b, err := h.svc.store.GetBadge(c.Request.Context(), c.Param("id"))
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

func (h *adminHandler) UpsertBadge(c *gin.Context) {
	var p badgeUpsertPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	id := strings.TrimSpace(p.ID)
	if pathID := strings.TrimSpace(c.Param("id")); pathID != "" {
		id = pathID
	}
	if id == "" {
		respondError(c, http.StatusBadRequest, "id required")
		return
	}
	b := &Badge{
		ID:          id,
		Title:       p.Title,
		Series:      p.Series,
		Type:        p.Type,
		StyleKey:    p.StyleKey,
		ImageURL:    p.ImageURL,
		Description: p.Description,
		SerialNo:    p.SerialNo,
		ReleasedAt:  p.ReleasedAt,
	}
	if cur, err := h.svc.store.GetBadge(c.Request.Context(), id); err == nil {
		b.CreatedAt = cur.CreatedAt
	}
	if err := h.svc.store.UpsertBadge(c.Request.Context(), b); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(c, b)
}

func (h *adminHandler) DeleteBadge(c *gin.Context) {
	if err := h.svc.store.DeleteBadge(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(c, gin.H{"ok": true})
}

func (h *adminHandler) UploadBadgeImage(c *gin.Context) {
	id := c.Param("id")
	cur, err := h.svc.store.GetBadge(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(c, http.StatusNotFound, "not found")
			return
		}
		respondError(c, http.StatusInternalServerError, "internal error")
		return
	}
	f, _, err := c.Request.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "file missing")
		return
	}
	defer f.Close()
	key, _, _, err := h.svc.IngestImage(c.Request.Context(), "badges/"+id, f)
	if err != nil {
		switch err {
		case ErrTooLarge:
			respondError(c, http.StatusRequestEntityTooLarge, "file too large")
		case ErrUnsupportedMedia:
			respondError(c, http.StatusUnsupportedMediaType, "unsupported media")
		default:
			respondError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	cur.ImageURL = key
	if err := h.svc.store.UpsertBadge(c.Request.Context(), cur); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(c, gin.H{"key": key})
}

func (h *adminHandler) ListPhotoRequests(c *gin.Context) {
	limit, offset := pageParams(c)
	items, total, err := h.svc.store.ListPhotoRequests(c.Request.Context(),
		c.Query("badgeId"), c.Query("status"), limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(c, gin.H{"items": items, "total": total})
}

type statusPayload struct {
	Status string `json:"status"`
}

type uploadPresignPayload struct {
	BadgeID     string `json:"badgeId"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Purpose     string `json:"purpose"`
}

func (h *adminHandler) PresignUpload(c *gin.Context) {
	var p uploadPresignPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	out, err := h.svc.PresignUpload(c.Request.Context(), p.BadgeID, p.FileName, p.ContentType, p.Purpose)
	if err != nil {
		switch {
		case errors.Is(err, ErrCOSNotConfigured):
			respondError(c, http.StatusServiceUnavailable, "cos not configured")
		default:
			respondError(c, http.StatusBadRequest, err.Error())
		}
		return
	}
	respondData(c, out)
}

type nfcWritePayload struct {
	BadgeID        string `json:"badgeId"`
	TagUID         string `json:"tagUid"`
	NDEFURL        string `json:"ndefUrl"`
	DeviceID       string `json:"deviceId"`
	WriteStatus    string `json:"writeStatus"`
	PhotoObjectKey string `json:"photoObjectKey"`
	WrittenAt      string `json:"writtenAt"`
}

func (h *adminHandler) CreateNFCWrite(c *gin.Context) {
	var p nfcWritePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	badgeID := strings.TrimSpace(p.BadgeID)
	if badgeID == "" {
		respondError(c, http.StatusBadRequest, "badgeId required")
		return
	}
	photoKey := strings.TrimSpace(p.PhotoObjectKey)
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
	w := &NFCWrite{
		BadgeID:        badgeID,
		TagUID:         strings.TrimSpace(p.TagUID),
		NDEFURL:        strings.TrimSpace(p.NDEFURL),
		DeviceID:       strings.TrimSpace(p.DeviceID),
		WriteStatus:    strings.TrimSpace(p.WriteStatus),
		PhotoObjectKey: photoKey,
		WrittenAt:      writtenAt,
	}
	if err := h.svc.store.InsertNFCWrite(c.Request.Context(), w); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(c, w)
}

func (h *adminHandler) UpdatePhotoStatus(c *gin.Context) {
	var p statusPayload
	if err := c.ShouldBindJSON(&p); err != nil || !validStatus(p.Status) {
		respondError(c, http.StatusBadRequest, "invalid status")
		return
	}
	if err := h.svc.store.UpdatePhotoStatus(c.Request.Context(), c.Param("id"), p.Status); err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(c, http.StatusNotFound, "not found")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(c, gin.H{"ok": true})
}

func (h *adminHandler) ListAutographRequests(c *gin.Context) {
	limit, offset := pageParams(c)
	items, total, err := h.svc.store.ListAutographRequests(c.Request.Context(),
		c.Query("badgeId"), c.Query("status"), limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(c, gin.H{"items": items, "total": total})
}

func (h *adminHandler) UpdateAutographStatus(c *gin.Context) {
	var p statusPayload
	if err := c.ShouldBindJSON(&p); err != nil || !validStatus(p.Status) {
		respondError(c, http.StatusBadRequest, "invalid status")
		return
	}
	if err := h.svc.store.UpdateAutographStatus(c.Request.Context(), c.Param("id"), p.Status); err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(c, http.StatusNotFound, "not found")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(c, gin.H{"ok": true})
}

func validStatus(s string) bool {
	switch s {
	case StatusNew, StatusHandled, StatusRejected:
		return true
	}
	return false
}
