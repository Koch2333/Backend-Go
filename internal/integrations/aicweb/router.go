package aicweb

import (
	"io"
	"os"
	"strconv"
	"strings"

	av "backend-go/internal/avatar"
	em "backend-go/internal/email"
	emenv "backend-go/internal/email/envinit"
	"backend-go/internal/integrations/aicweb/envinit"
	"backend-go/internal/integrations/msconsent"

	"github.com/gin-gonic/gin"
)

// mountStaticOnce 把 dir 暴露到 prefix；该前缀已被注册过就跳过，
// 防止 avatar 模块和 aicweb 模块互相挂同一个静态目录时 gin 路由冲突。
func mountStaticOnce(engine *gin.Engine, prefix, dir string) {
	wildcard := prefix + "/*filepath"
	for _, ri := range engine.Routes() {
		if ri.Method == "GET" && ri.Path == wildcard {
			return
		}
	}
	engine.StaticFS(prefix, gin.Dir(dir, false))
}

// mediaAdapter wraps avatar.Service to implement MediaUploader.
type mediaAdapter struct{ svc *av.Service }

func (m *mediaAdapter) Upload(r io.Reader) (string, error) {
	_, _, url, err := m.svc.ProcessAndStore(r)
	return url, err
}

func newBannerService() (*av.Service, error) {
	dir := envOr("BANNER_DIR", "assets/banner")
	urlp := envOr("BANNER_URL_PREFIX", "/assets/banner")
	maxMB := envIntOr("BANNER_MAX_MB", 15)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &av.Service{
		Dir:      dir,
		URLPrefix: urlp,
		MaxBytes: int64(maxMB) * (1 << 20),
	}, nil
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// Mount 把所有路由挂到传入的 RouterGroup 上。
func Mount(engine *gin.Engine, r *gin.RouterGroup) {
	var svc Service
	if s, err := NewServiceSQLiteFromEnv(); err == nil {
		svc = s
	} else {
		svc = NewServiceMemory()
	}

	ts := NewTurnstileFromEnv()

	fs, err := NewFormServiceFromEnv()
	if err != nil {
		panic("failed to init sqlite form service: " + err.Error())
	}

	emenv.Init()
	sender := em.NewSenderFromEnv()
	notify := NewEmailActivationNotifierFromEnv(sender)

	// Avatar uploader (reuses existing avatar module service).
	avtSvc, _ := av.NewServiceFromEnv()
	var avt MediaUploader
	if avtSvc != nil {
		mountStaticOnce(engine, avtSvc.URLPrefix, avtSvc.Dir)
		avt = &mediaAdapter{svc: avtSvc}
	}

	// Banner uploader (separate dir/size limits).
	bnrSvc, _ := newBannerService()
	var bnr MediaUploader
	if bnrSvc != nil {
		mountStaticOnce(engine, bnrSvc.URLPrefix, bnrSvc.Dir)
		bnr = &mediaAdapter{svc: bnrSvc}
	}

	h := NewHandler(svc, ts, fs, notify, avt, bnr)

	// 公共路由
	r.POST("/user/register", h.Register)
	r.POST("/user/login", h.Login)
	r.GET("/user/activate", h.Activate)
	r.GET("/user/profiles", h.ListProfiles)
	r.GET("/user/profile/:username", h.GetPublicProfile)

	// 受保护路由
	prv := r.Group("", AuthRequired(svc))
	{
		prv.GET("/user/profile", h.Profile)
		prv.PUT("/user/me/profile", h.UpdateMyProfile)
		prv.PUT("/user/me/avatar", h.UploadAvatar)
		prv.PUT("/user/me/banner", h.UploadBanner)
		prv.POST("/user/form", h.SubmitForm)
		prv.GET("/user/form", h.ListMyForms)
	}
}

func Attach(engine *gin.Engine) {
	envinit.Init()
	base := os.Getenv("AICWEB_BASE_PREFIX")
	if base == "" {
		base = "/api/aicweb"
	}
	AttachTo(engine, base)
}

func AttachTo(engine *gin.Engine, prefix string) {
	envinit.Init()
	if prefix == "" {
		prefix = "/api/aicweb"
	}
	grp := engine.Group(prefix)
	Mount(engine, grp)
	msconsent.Attach(engine)
}
