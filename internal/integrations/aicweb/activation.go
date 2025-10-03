package aicweb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	em "backend-go/internal/email"
)

// ActivationNotifier：aicweb 需要的邮件策略
type ActivationNotifier interface {
	SendActivation(to, token string) error
}

// EmailActivationNotifier：用 email.Sender 适配，并支持把 token 落地到本地文件
type EmailActivationNotifier struct {
	sender    em.Sender
	baseURL   string // 例如 https://your-domain.com/api/aicweb/user/activate
	debugFile string // 例如 databases/aicweb/activation_tokens.debug.log
}

func NewEmailActivationNotifierFromEnv(sender em.Sender) *EmailActivationNotifier {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("AICWEB_ACTIVATION_BASE_URL")), "/")
	if base == "" {
		base = "http://localhost:8080/api/aicweb/user/activate"
	}
	debug := strings.TrimSpace(os.Getenv("AICWEB_ACTIVATION_DEBUG_FILE"))
	if debug == "" {
		debug = "databases/aicweb/activation_tokens.debug.log"
	}
	// 确保目录存在（写文件时也会再确保一次，这里提前做一遍）
	_ = os.MkdirAll(filepath.Dir(debug), 0o755)

	return &EmailActivationNotifier{
		sender:    sender,
		baseURL:   base,
		debugFile: debug,
	}
}

func (n *EmailActivationNotifier) SendActivation(to, token string) error {
	if n == nil {
		return nil
	}
	link := fmt.Sprintf("%s?token=%s", n.baseURL, token)

	// 1) 落地到本地文件（时间、邮箱、token、完整链接）
	_ = appendLine(n.debugFile, fmt.Sprintf("%s\t%s\t%s\t%s\n",
		time.Now().Format(time.RFC3339), to, token, link))

	// 2) 通过策略真正发送邮件（none/log/smtp）
	if n.sender != nil {
		html := fmt.Sprintf(`<p>你好！请点击以下链接激活你的账号：</p><p><a href="%s">%s</a></p>`, link, link)
		_ = n.sender.Send(to, "激活你的账号", html, "请在浏览器打开链接："+link)
	}
	return nil
}

func appendLine(path, line string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.WriteString(line)
	return err
}
