package service

import (
	"errors"
	"fmt"
	"net/smtp"
	"strings"

	"constructor-script-backend/internal/config"
)

type EmailService struct {
	config *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{config: cfg}
}

func (s *EmailService) Enabled() bool {
	if s == nil || s.config == nil {
		return false
	}

	host := strings.TrimSpace(s.config.SMTPHost)
	username := strings.TrimSpace(s.config.SMTPUsername)
	password := strings.TrimSpace(s.config.SMTPPassword)

	return s.config.EnableEmail && host != "" && username != "" && password != ""
}

func (s *EmailService) Send(to, subject, body string) error {
	if s == nil || !s.Enabled() {
		return errors.New("email service is disabled or not configured")
	}

	host := strings.TrimSpace(s.config.SMTPHost)
	port := strings.TrimSpace(s.config.SMTPPort)
	if port == "" {
		port = "587"
	}

	from := strings.TrimSpace(s.config.SMTPFrom)
	if from == "" {
		from = "noreply@" + host
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	auth := smtp.PlainAuth("", strings.TrimSpace(s.config.SMTPUsername), strings.TrimSpace(s.config.SMTPPassword), host)

	var builder strings.Builder
	headers := map[string]string{
		"From":         from,
		"To":           strings.TrimSpace(to),
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}

	for key, value := range headers {
		builder.WriteString(key)
		builder.WriteString(": ")
		builder.WriteString(value)
		builder.WriteString("\r\n")
	}

	builder.WriteString("\r\n")
	builder.WriteString(body)

	return smtp.SendMail(addr, auth, from, []string{to}, []byte(builder.String()))
}
