package utils

import (
	"gopkg.in/gomail.v2"
	"mirrors_status/internal/config"
)

func SendMail(subject, content string, tos... string) error {
	dialer := configs.GetMailDialer()
	m := gomail.NewMessage()
	m.SetHeader("From", dialer.Username)
	m.SetHeader("Subject", subject)
	m.SetHeader("To", tos...)
	m.SetBody("text/html", "<h1>" + content + "</h1>")
	return dialer.DialAndSend(m)
}
