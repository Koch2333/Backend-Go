package email

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Sender 是邮件发送策略接口
type Sender interface {
	Send(to, subject, htmlBody, textBody string) error
	Name() string
}

// NewSenderFromEnv 根据 EMAIL_STRATEGY 选择策略：smtp | log | none
func NewSenderFromEnv() Sender {
	strategy := strings.ToLower(strings.TrimSpace(os.Getenv("EMAIL_STRATEGY")))
	switch strategy {
	case "smtp":
		return newSMTPSenderFromEnv()
	case "log":
		return logSender{}
	default:
		return noneSender{}
	}
}

// ---------- none 策略（禁用邮件，什么也不做） ----------
type noneSender struct{}

func (noneSender) Send(to, subject, htmlBody, textBody string) error { return nil }
func (noneSender) Name() string                                      { return "none" }

// ---------- log 策略（开发态打印） ----------
type logSender struct{}

func (logSender) Send(to, subject, htmlBody, textBody string) error {
	fmt.Printf("[email/log] to=%s subject=%q html=%dB text=%dB\n", to, subject, len(htmlBody), len(textBody))
	return nil
}
func (logSender) Name() string { return "log" }

// ---------- smtp 策略（在 strategy_smtp.go） ----------
type smtpSender struct {
	host, user, pass, from string
	port                   int
}

func newSMTPSenderFromEnv() Sender {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	port, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("SMTP_PORT")))
	user := strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	pass := strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if host == "" || port == 0 || user == "" || pass == "" || from == "" {
		return noneSender{}
	}
	return smtpSender{host: host, port: port, user: user, pass: pass, from: from}
}
