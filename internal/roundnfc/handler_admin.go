package roundnfc

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type adminHandler struct {
	svc       *Service
	apiPrefix string
}

func newAdminHandler(svc *Service, apiPrefix string) *adminHandler {
	return &adminHandler{svc: svc, apiPrefix: strings.TrimRight(apiPrefix, "/")}
}

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
	for i := range items {
		items[i] = h.svc.PublicBadge(c.Request.Context(), &items[i], h.apiPrefix)
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
	out := h.svc.PublicBadge(c.Request.Context(), b, h.apiPrefix)
	respondData(c, out)
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
	styleKey := strings.TrimSpace(p.StyleKey)
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
		ID:          id,
		Title:       p.Title,
		Series:      p.Series,
		Type:        p.Type,
		StyleKey:    styleKey,
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
	log.Printf("[roundnfc/admin] upsert badge id=%q styleKey=%q ip=%s", b.ID, b.StyleKey, c.ClientIP())
	respondData(c, b)
}

func (h *adminHandler) DeleteBadge(c *gin.Context) {
	if err := h.svc.store.DeleteBadge(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[roundnfc/admin] delete badge id=%q ip=%s", c.Param("id"), c.ClientIP())
	respondData(c, gin.H{"ok": true})
}

func (h *adminHandler) ListStyleTemplates(c *gin.Context) {
	items, err := h.svc.store.ListBadgeStyleTemplates(c.Request.Context(), false)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range items {
		items[i] = h.svc.PublicStyleTemplate(c.Request.Context(), items[i], h.apiPrefix)
	}
	respondData(c, gin.H{"items": items})
}

type styleTemplatePayload struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	Payload     any    `json:"payload"`
	Enabled     *bool  `json:"enabled"`
}

func (h *adminHandler) UpsertStyleTemplate(c *gin.Context) {
	var p styleTemplatePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	key := strings.TrimSpace(p.Key)
	if pathKey := strings.TrimSpace(c.Param("key")); pathKey != "" {
		key = pathKey
	}
	if key == "" {
		respondError(c, http.StatusBadRequest, "key required")
		return
	}
	label := strings.TrimSpace(p.Label)
	if label == "" {
		respondError(c, http.StatusBadRequest, "label required")
		return
	}
	var cur *BadgeStyleTemplate
	if existing, err := h.svc.store.GetBadgeStyleTemplate(c.Request.Context(), key); err == nil {
		cur = existing
	}
	payload := []byte("{}")
	if p.Payload != nil {
		var err error
		payload, err = jsonMarshalRaw(p.Payload)
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid payload")
			return
		}
	} else if cur != nil {
		payload = cur.Payload
	}
	enabled := true
	if p.Enabled != nil {
		enabled = *p.Enabled
	} else if cur != nil {
		enabled = cur.Enabled
	}
	t := &BadgeStyleTemplate{
		Key:         key,
		Label:       label,
		Description: strings.TrimSpace(p.Description),
		ImageURL:    strings.TrimSpace(p.ImageURL),
		Payload:     payload,
		Enabled:     enabled,
	}
	if cur != nil {
		t.CreatedAt = cur.CreatedAt
	}
	if err := h.svc.store.UpsertBadgeStyleTemplate(c.Request.Context(), t); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[roundnfc/admin] upsert style template key=%q label=%q enabled=%t imageUrl=%q ip=%s",
		t.Key, t.Label, t.Enabled, t.ImageURL, c.ClientIP())
	respondData(c, t)
}

func (h *adminHandler) UploadStyleTemplateImage(c *gin.Context) {
	key := strings.TrimSpace(c.Param("key"))
	if key == "" {
		respondError(c, http.StatusBadRequest, "key required")
		return
	}
	cur, err := h.svc.store.GetBadgeStyleTemplate(c.Request.Context(), key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(c, http.StatusNotFound, "not found")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	f, _, err := c.Request.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "file missing")
		return
	}
	defer f.Close()
	imageKey, _, _, err := h.svc.IngestImage(c.Request.Context(), "style-templates/"+key, f)
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
	cur.ImageURL = imageKey
	if err := h.svc.store.UpsertBadgeStyleTemplate(c.Request.Context(), cur); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[roundnfc/admin] upload style template image key=%q imageUrl=%q ip=%s", key, imageKey, c.ClientIP())
	respondData(c, gin.H{"key": imageKey, "item": cur})
}

func (h *adminHandler) DeleteStyleTemplate(c *gin.Context) {
	if err := h.svc.store.DeleteBadgeStyleTemplate(c.Request.Context(), c.Param("key")); err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(c, http.StatusNotFound, "not found")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[roundnfc/admin] delete style template key=%q ip=%s", c.Param("key"), c.ClientIP())
	respondData(c, gin.H{"ok": true})
}

func jsonMarshalRaw(v any) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (h *adminHandler) ListAppTokens(c *gin.Context) {
	items, err := h.svc.store.ListAppTokens(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(c, gin.H{"items": items})
}

type appTokenCreatePayload struct {
	Name    string `json:"name"`
	ApiBase string `json:"apiBase"`
}

func (h *adminHandler) CreateAppToken(c *gin.Context) {
	var p appTokenCreatePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		respondError(c, http.StatusBadRequest, "name required")
		return
	}
	apiBase, err := normalizePairingAPIBase(p.ApiBase)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid apiBase")
		return
	}
	if apiBase == "" {
		apiBase = inferAPIBase(c.Request)
	}
	plain, err := newAppTokenPlain()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "token generate failed")
		return
	}
	item := &AppToken{
		ID:          uuid.NewString(),
		Name:        name,
		TokenPrefix: appTokenPrefix(plain),
		Enabled:     true,
	}
	if err := h.svc.store.InsertAppToken(c.Request.Context(), item, appTokenHash(plain)); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	pairing := h.appPairingConfig(name, apiBase, plain)
	log.Printf("[roundnfc/admin] create app token id=%q name=%q prefix=%q apiBase=%q ip=%s",
		item.ID, item.Name, item.TokenPrefix, apiBase, c.ClientIP())
	respondData(c, gin.H{"item": item, "token": plain, "pairing": pairing})
}

type appTokenUpdatePayload struct {
	Enabled bool `json:"enabled"`
}

func (h *adminHandler) UpdateAppToken(c *gin.Context) {
	var p appTokenUpdatePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.svc.store.SetAppTokenEnabled(c.Request.Context(), c.Param("id"), p.Enabled); err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(c, http.StatusNotFound, "not found")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[roundnfc/admin] update app token id=%q enabled=%t ip=%s", c.Param("id"), p.Enabled, c.ClientIP())
	respondData(c, gin.H{"ok": true})
}

func (h *adminHandler) DeleteAppToken(c *gin.Context) {
	if err := h.svc.store.DeleteAppToken(c.Request.Context(), c.Param("id")); err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(c, http.StatusNotFound, "not found")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[roundnfc/admin] delete app token id=%q ip=%s", c.Param("id"), c.ClientIP())
	respondData(c, gin.H{"ok": true})
}

func (h *adminHandler) appPairingConfig(name, apiBase, token string) AppPairingConfig {
	prefix := h.apiPrefix
	if prefix == "" {
		prefix = "/api/roundnfc"
	}
	return AppPairingConfig{
		Protocol:    "roundnfc-writer",
		Version:     1,
		Name:        name,
		ApiBase:     apiBase,
		ApiPrefix:   prefix,
		TokenHeader: appTokenHeader,
		Token:       token,
		Endpoints: map[string]string{
			"listStyleTemplates": prefix + "/app/style-templates",
			"listBadges":         prefix + "/app/badges",
			"getBadge":           prefix + "/app/badges/{id}",
			"upsertBadge":        prefix + "/app/badges",
			"presignUpload":      prefix + "/app/uploads/presign",
			"createWrite":        prefix + "/app/nfc-writes",
			"presignCoserPhoto":  prefix + "/app/badges/{id}/coser-photo/presign",
			"upsertCoserBinding": prefix + "/app/badges/{id}/coser-binding",
			"getCoserBinding":    prefix + "/app/badges/{id}/coser-binding",
		},
		CreatedAt: time.Now().UTC(),
	}
}

func normalizePairingAPIBase(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	for strings.HasSuffix(raw, "/") {
		raw = strings.TrimSuffix(raw, "/")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("invalid apiBase")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func inferAPIBase(r *http.Request) string {
	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		proto = "http"
		if r.TLS != nil {
			proto = "https"
		}
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	if host == "" {
		return ""
	}
	return proto + "://" + host
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
	log.Printf("[roundnfc/admin] upload badge image badgeId=%q imageUrl=%q ip=%s", id, key, c.ClientIP())
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
		log.Printf("[roundnfc/admin] presign upload failed badgeId=%q purpose=%q fileName=%q contentType=%q ip=%s err=%v",
			p.BadgeID, p.Purpose, p.FileName, p.ContentType, c.ClientIP(), err)
		switch {
		case errors.Is(err, ErrCOSNotConfigured):
			respondError(c, http.StatusServiceUnavailable, "cos not configured")
		default:
			respondError(c, http.StatusBadRequest, err.Error())
		}
		return
	}
	log.Printf("[roundnfc/admin] presign upload badgeId=%q purpose=%q objectKey=%q contentType=%q expiresIn=%d ip=%s",
		p.BadgeID, p.Purpose, out.ObjectKey, p.ContentType, out.ExpiresIn, c.ClientIP())
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
	log.Printf("[roundnfc/admin] create nfc write id=%q badgeId=%q status=%q tagUid=%q deviceId=%q photoObjectKey=%q ip=%s",
		w.ID, w.BadgeID, w.WriteStatus, w.TagUID, w.DeviceID, w.PhotoObjectKey, c.ClientIP())
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
