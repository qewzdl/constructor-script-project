package service

import (
	"errors"
	"fmt"
	"net/smtp"
	"strings"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/repository"
)

type EmailService struct {
	config      *config.Config
	settingRepo repository.SettingRepository
}

type emailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func NewEmailService(cfg *config.Config, settingRepo repository.SettingRepository) *EmailService {
	return &EmailService{
		config:      cfg,
		settingRepo: settingRepo,
	}
}

func (s *EmailService) Enabled() bool {
	if s.config != nil && !s.config.EnableEmail {
		return false
	}
	cfg := s.resolveConfig()
	return cfg.Host != "" && cfg.Username != "" && cfg.Password != ""
}

func (s *EmailService) Send(to, subject, body string) error {
	if s == nil || !s.Enabled() {
		return errors.New("email service is disabled or not configured")
	}

	cfg := s.resolveConfig()

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	var builder strings.Builder
	headers := map[string]string{
		"From":         cfg.From,
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

	return smtp.SendMail(addr, auth, cfg.From, []string{to}, []byte(builder.String()))
}

func (s *EmailService) resolveConfig() emailConfig {
	result := emailConfig{Port: "587"}
	if s == nil {
		return result
	}

	if s.config != nil {
		result.Host = strings.TrimSpace(s.config.SMTPHost)
		result.Port = strings.TrimSpace(s.config.SMTPPort)
		result.Username = strings.TrimSpace(s.config.SMTPUsername)
		result.Password = strings.TrimSpace(s.config.SMTPPassword)
		result.From = strings.TrimSpace(s.config.SMTPFrom)
	}

	if s.settingRepo != nil {
		if value := s.readSetting(settingKeySMTPHost); value != "" {
			result.Host = value
		}
		if value := s.readSetting(settingKeySMTPPort); value != "" {
			result.Port = value
		}
		if value := s.readSetting(settingKeySMTPUsername); value != "" {
			result.Username = value
		}
		if value := s.readSetting(settingKeySMTPPassword); value != "" {
			result.Password = value
		}
		if value := s.readSetting(settingKeySMTPFrom); value != "" {
			result.From = value
		}
	}

	if strings.TrimSpace(result.Port) == "" {
		result.Port = "587"
	}
	if strings.TrimSpace(result.From) == "" && result.Host != "" {
		result.From = "noreply@" + result.Host
	}

	return result
}

func (s *EmailService) readSetting(key string) string {
	if s == nil || s.settingRepo == nil {
		return ""
	}

	setting, err := s.settingRepo.Get(key)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(setting.Value)
}
