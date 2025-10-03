package aicweb

import (
	"os"

	em "backend-go/internal/email"
	emenv "backend-go/internal/email/envinit"
	"backend-go/internal/integrations/aicweb/envinit"

	"github.com/gin-gonic/gin"
)

// 只负责把 handler 挂到传入分组上
func Mount(r *gin.RouterGroup) {
	// 账号服务：优先 SQLite，失败回退内存实现
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

	// 邮件策略（独立 email 模块）
	emenv.Init()
	sender := em.NewSenderFromEnv()
	notify := NewEmailActivationNotifierFromEnv(sender)

	h := NewHandler(svc, ts, fs, notify)

	// 公共路由
	r.POST("/user/register", h.Register)
	r.POST("/user/login", h.Login)
	r.GET("/user/activate", h.Activate)

	// 受保护路由
	prv := r.Group("", AuthRequired(svc))
	{
		prv.GET("/user/profile", h.Profile)
		prv.POST("/user/form", h.SubmitForm)
		prv.GET("/user/form", h.ListMyForms)
	}
}

// 兼容旧用法：读取环境变量前缀并挂载
func Attach(engine *gin.Engine) {
	envinit.Init()
	base := os.Getenv("AICWEB_BASE_PREFIX")
	if base == "" {
		base = "/api/aicweb"
	}
	AttachTo(engine, base)
}

// ★ 提供给自动挂载框架使用：可指定任意前缀
func AttachTo(engine *gin.Engine, prefix string) {
	envinit.Init()
	if prefix == "" {
		prefix = "/api/aicweb"
	}
	grp := engine.Group(prefix)
	Mount(grp)
}
