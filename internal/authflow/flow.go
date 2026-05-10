package authflow

import (
	"net/http"
	"strings"
	"time"

	"backend-go/internal/auth"

	"github.com/gin-gonic/gin"
)

// Flow handles admin authentication: password+TOTP login, passkey ceremonies, and MFA management.
type Flow struct {
	cfg  Config
	pool waPool
}

// New creates a new Flow.
func New(cfg Config) *Flow {
	f := &Flow{cfg: cfg}
	f.pool.sess = make(map[string]*waChallenge)
	return f
}

// Mount attaches all auth routes to the /admin RouterGroup.
// Unauthenticated: POST /login, POST /webauthn/login/begin, POST /webauthn/login/finish.
// Authenticated: GET /me, GET /totp/status, POST /totp/setup, POST /totp/enable, DELETE /totp,
//   POST /webauthn/register/begin, POST /webauthn/register/finish,
//   GET /webauthn/credentials, DELETE /webauthn/credentials/:id.
func (f *Flow) Mount(admin *gin.RouterGroup) {
	admin.POST("/login", f.handleLogin)
	admin.POST("/webauthn/login/begin", f.handleWALoginBegin)
	admin.POST("/webauthn/login/finish", f.handleWALoginFinish)

	g := admin.Group("", auth.Required(f.cfg.JWTSecret))
	g.GET("/me", f.handleMe)
	g.GET("/totp/status", f.handleTOTPStatus)
	g.POST("/totp/setup", f.handleTOTPSetup)
	g.POST("/totp/enable", f.handleTOTPEnable)
	g.DELETE("/totp", f.handleTOTPDisable)
	g.POST("/webauthn/register/begin", f.handleWARegisterBegin)
	g.POST("/webauthn/register/finish", f.handleWARegisterFinish)
	g.GET("/webauthn/credentials", f.handleWAListCredentials)
	g.DELETE("/webauthn/credentials/:id", f.handleWADeleteCredential)
}

// ---------- login ----------

type loginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TOTPCode string `json:"totpCode"`
}

func (f *Flow) handleLogin(c *gin.Context) {
	var p loginPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		flowFail(c, http.StatusBadRequest, "invalid body")
		return
	}
	if f.cfg.AdminPasswordHash == "" || len(f.cfg.JWTSecret) < 16 {
		flowFail(c, http.StatusServiceUnavailable, "admin not configured")
		return
	}
	if p.Username != f.cfg.AdminUsername || !auth.VerifyPassword(f.cfg.AdminPasswordHash, p.Password) {
		flowFail(c, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if f.cfg.Store != nil {
		secret, enabled, _ := f.cfg.Store.GetTOTP(p.Username)
		if enabled {
			if strings.TrimSpace(p.TOTPCode) == "" {
				c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": gin.H{"needsTOTP": true}})
				return
			}
			if !VerifyTOTP(secret, p.TOTPCode) {
				flowFail(c, http.StatusUnauthorized, "invalid TOTP code")
				return
			}
		}
	}
	tok, exp, err := auth.IssueToken(f.cfg.JWTSecret, p.Username, f.cfg.JWTTTL)
	if err != nil {
		flowFail(c, http.StatusInternalServerError, "token error")
		return
	}
	flowOK(c, gin.H{"token": tok, "expiresAt": exp.Format(time.RFC3339), "username": p.Username})
}

func (f *Flow) handleMe(c *gin.Context) {
	flowOK(c, gin.H{"username": c.GetString(auth.ContextKeySubject)})
}

// ---------- TOTP ----------

func (f *Flow) handleTOTPStatus(c *gin.Context) {
	username := c.GetString(auth.ContextKeySubject)
	_, enabled, _ := f.cfg.Store.GetTOTP(username)
	flowOK(c, gin.H{"enabled": enabled})
}

func (f *Flow) handleTOTPSetup(c *gin.Context) {
	username := c.GetString(auth.ContextKeySubject)
	secret, err := genTOTPSecret()
	if err != nil {
		flowFail(c, http.StatusInternalServerError, "generate secret failed")
		return
	}
	issuer := f.cfg.TOTPIssuer
	if issuer == "" {
		issuer = "Backend"
	}
	if err := f.cfg.Store.SetTOTP(username, secret, false); err != nil {
		flowFail(c, http.StatusInternalServerError, "save failed")
		return
	}
	flowOK(c, gin.H{"uri": TOTPProvisioningURI(issuer, username, secret), "secret": secret})
}

type totpEnablePayload struct {
	Code string `json:"code"`
}

func (f *Flow) handleTOTPEnable(c *gin.Context) {
	username := c.GetString(auth.ContextKeySubject)
	var p totpEnablePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		flowFail(c, http.StatusBadRequest, "invalid body")
		return
	}
	secret, _, err := f.cfg.Store.GetTOTP(username)
	if err != nil || secret == "" {
		flowFail(c, http.StatusBadRequest, "no pending TOTP setup")
		return
	}
	if !VerifyTOTP(secret, p.Code) {
		flowFail(c, http.StatusBadRequest, "invalid code")
		return
	}
	if err := f.cfg.Store.SetTOTP(username, secret, true); err != nil {
		flowFail(c, http.StatusInternalServerError, "save failed")
		return
	}
	flowOK(c, gin.H{"ok": true})
}

func (f *Flow) handleTOTPDisable(c *gin.Context) {
	username := c.GetString(auth.ContextKeySubject)
	if err := f.cfg.Store.SetTOTP(username, "", false); err != nil {
		flowFail(c, http.StatusInternalServerError, "save failed")
		return
	}
	flowOK(c, gin.H{"ok": true})
}

// ---------- WebAuthn registration ----------

func (f *Flow) handleWARegisterBegin(c *gin.Context) {
	username := c.GetString(auth.ContextKeySubject)
	creds, _ := f.cfg.Store.GetCredentials(username)
	existingIDs := make([][]byte, len(creds))
	for i, cr := range creds {
		raw, _ := waB64Decode(cr.ID)
		existingIDs[i] = raw
	}
	opts, err := beginRegistration(&f.pool, &f.cfg, username, existingIDs)
	if err != nil {
		flowFail(c, http.StatusInternalServerError, "begin registration failed")
		return
	}
	flowOK(c, opts)
}

type waRegisterFinishPayload struct {
	SessionID  string                     `json:"sessionId"`
	Name       string                     `json:"name"`
	Credential CredentialCreationResponse `json:"credential"`
}

func (f *Flow) handleWARegisterFinish(c *gin.Context) {
	var p waRegisterFinishPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		flowFail(c, http.StatusBadRequest, "invalid body")
		return
	}
	cred, err := finishRegistration(&f.pool, &f.cfg, p.SessionID, &p.Credential)
	if err != nil {
		flowFail(c, http.StatusBadRequest, err.Error())
		return
	}
	cred.Name = p.Name
	if cred.Name == "" {
		cred.Name = "Passkey"
	}
	if err := f.cfg.Store.SaveCredential(cred); err != nil {
		flowFail(c, http.StatusInternalServerError, "save failed")
		return
	}
	flowOK(c, gin.H{"ok": true, "id": cred.ID})
}

func (f *Flow) handleWAListCredentials(c *gin.Context) {
	username := c.GetString(auth.ContextKeySubject)
	list, err := f.cfg.Store.ListCredentials(username)
	if err != nil {
		flowFail(c, http.StatusInternalServerError, err.Error())
		return
	}
	flowOK(c, gin.H{"items": list})
}

func (f *Flow) handleWADeleteCredential(c *gin.Context) {
	username := c.GetString(auth.ContextKeySubject)
	if err := f.cfg.Store.DeleteCredential(username, c.Param("id")); err != nil {
		flowFail(c, http.StatusInternalServerError, err.Error())
		return
	}
	flowOK(c, gin.H{"ok": true})
}

// ---------- WebAuthn login ----------

type waLoginBeginPayload struct {
	Username string `json:"username"`
}

func (f *Flow) handleWALoginBegin(c *gin.Context) {
	var p waLoginBeginPayload
	_ = c.ShouldBindJSON(&p)
	if p.Username == "" {
		p.Username = f.cfg.AdminUsername
	}
	if f.cfg.Store == nil {
		flowFail(c, http.StatusBadRequest, "passkeys not configured")
		return
	}
	creds, err := f.cfg.Store.GetCredentials(p.Username)
	if err != nil || len(creds) == 0 {
		flowFail(c, http.StatusBadRequest, "no passkeys registered")
		return
	}
	opts, err := beginLogin(&f.pool, &f.cfg, p.Username, creds)
	if err != nil {
		flowFail(c, http.StatusInternalServerError, err.Error())
		return
	}
	flowOK(c, opts)
}

type waLoginFinishPayload struct {
	SessionID  string                      `json:"sessionId"`
	Credential CredentialAssertionResponse `json:"credential"`
}

func (f *Flow) handleWALoginFinish(c *gin.Context) {
	var p waLoginFinishPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		flowFail(c, http.StatusBadRequest, "invalid body")
		return
	}
	cred, err := finishLogin(&f.pool, &f.cfg, p.SessionID, &p.Credential, f.cfg.Store.GetCredentials)
	if err != nil {
		flowFail(c, http.StatusUnauthorized, err.Error())
		return
	}
	_ = f.cfg.Store.UpdateCounter(cred.ID, cred.Counter)
	tok, exp, err := auth.IssueToken(f.cfg.JWTSecret, cred.Username, f.cfg.JWTTTL)
	if err != nil {
		flowFail(c, http.StatusInternalServerError, "token error")
		return
	}
	flowOK(c, gin.H{"token": tok, "expiresAt": exp.Format(time.RFC3339), "username": cred.Username})
}

// ---------- response helpers ----------

func flowOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": data})
}

func flowFail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"code": status, "message": msg, "data": nil})
}
