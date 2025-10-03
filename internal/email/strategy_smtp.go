package email

import (
	gomail "gopkg.in/gomail.v2"
)

func (s smtpSender) Send(to, subject, htmlBody, textBody string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", s.from)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	if htmlBody != "" {
		msg.SetBody("text/html", htmlBody)
	} else {
		msg.SetBody("text/plain", textBody)
	}
	d := gomail.NewDialer(s.host, s.port, s.user, s.pass)
	return d.DialAndSend(msg)
}

func (s smtpSender) Name() string { return "smtp" }
