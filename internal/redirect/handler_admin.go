package redirect

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type adminHandler struct{ svc *Service }

func newAdminHandler(svc *Service) *adminHandler { return &adminHandler{svc: svc} }

func adminOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": data})
}

func adminFail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"code": status, "message": msg, "data": nil})
}

func pageParams(c *gin.Context) (limit, offset int) {
	limit = 100
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}

// ----- rules -----

type ruleDTO struct {
	Name      string `json:"name"`
	TargetURL string `json:"targetUrl"`
	Enabled   bool   `json:"enabled"`
	UpdatedAt string `json:"updatedAt"`
}

func (h *adminHandler) ListRules(c *gin.Context) {
	limit, offset := pageParams(c)
	rows, total, err := h.svc.Store.ListRules(strings.TrimSpace(c.Query("q")), limit, offset)
	if err != nil {
		adminFail(c, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]ruleDTO, 0, len(rows))
	for _, r := range rows {
		items = append(items, ruleDTO{
			Name: r.Name, TargetURL: r.TargetURL, Enabled: r.Enabled,
			UpdatedAt: r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	adminOK(c, gin.H{"items": items, "total": total})
}

type ruleUpsertPayload struct {
	Name      string `json:"name"`
	TargetURL string `json:"targetUrl"`
	Enabled   bool   `json:"enabled"`
}

func (h *adminHandler) UpsertRule(c *gin.Context) {
	var p ruleUpsertPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		adminFail(c, http.StatusBadRequest, "invalid body")
		return
	}
	name := strings.TrimSpace(p.Name)
	if pathName := strings.TrimSpace(c.Param("name")); pathName != "" {
		name = pathName
	}
	if name == "" {
		adminFail(c, http.StatusBadRequest, "name required")
		return
	}
	target := strings.TrimSpace(p.TargetURL)
	if target == "" {
		adminFail(c, http.StatusBadRequest, "targetUrl required")
		return
	}
	if err := h.svc.UpsertRule(name, target, p.Enabled); err != nil {
		adminFail(c, http.StatusInternalServerError, err.Error())
		return
	}
	adminOK(c, gin.H{"ok": true})
}

func (h *adminHandler) DeleteRule(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		adminFail(c, http.StatusBadRequest, "name required")
		return
	}
	if err := h.svc.Store.DeleteRule(name); err != nil {
		adminFail(c, http.StatusInternalServerError, err.Error())
		return
	}
	adminOK(c, gin.H{"ok": true})
}

// ----- nfc cards -----

type cardDTO struct {
	HWID         string `json:"hwid"`
	IsRegistered bool   `json:"isRegistered"`
	UserID       string `json:"userId"`
	UpdatedAt    string `json:"updatedAt"`
}

func (h *adminHandler) ListCards(c *gin.Context) {
	limit, offset := pageParams(c)
	rows, total, err := h.svc.Store.ListCards(strings.TrimSpace(c.Query("q")), limit, offset)
	if err != nil {
		adminFail(c, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]cardDTO, 0, len(rows))
	for _, r := range rows {
		items = append(items, cardDTO{
			HWID: r.HWID, IsRegistered: r.IsRegistered, UserID: r.UserID,
			UpdatedAt: r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	adminOK(c, gin.H{"items": items, "total": total})
}

type cardUpsertPayload struct {
	HWID         string `json:"hwid"`
	IsRegistered bool   `json:"isRegistered"`
	UserID       string `json:"userId"`
}

func (h *adminHandler) UpsertCard(c *gin.Context) {
	var p cardUpsertPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		adminFail(c, http.StatusBadRequest, "invalid body")
		return
	}
	hwid := strings.TrimSpace(p.HWID)
	if pathHwid := strings.TrimSpace(c.Param("hwid")); pathHwid != "" {
		hwid = pathHwid
	}
	if hwid == "" {
		adminFail(c, http.StatusBadRequest, "hwid required")
		return
	}
	if err := h.svc.UpsertCard(hwid, p.IsRegistered, strings.TrimSpace(p.UserID)); err != nil {
		adminFail(c, http.StatusInternalServerError, err.Error())
		return
	}
	adminOK(c, gin.H{"ok": true})
}

func (h *adminHandler) DeleteCard(c *gin.Context) {
	hwid := strings.TrimSpace(c.Param("hwid"))
	if hwid == "" {
		adminFail(c, http.StatusBadRequest, "hwid required")
		return
	}
	if err := h.svc.Store.DeleteCard(hwid); err != nil {
		adminFail(c, http.StatusInternalServerError, err.Error())
		return
	}
	adminOK(c, gin.H{"ok": true})
}
