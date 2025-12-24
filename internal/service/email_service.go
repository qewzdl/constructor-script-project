package service

import (
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/logger"
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
	if s == nil {
		return errors.New("email service is disabled or not configured")
	}

	cfg := s.resolveConfig()
	if s.config != nil && !s.config.EnableEmail {
		logger.Warn("Email service disabled by config flag", map[string]interface{}{
			"enable_email":      s.config.EnableEmail,
			"smtp_host_set":     strings.TrimSpace(cfg.Host) != "",
			"smtp_port_set":     strings.TrimSpace(cfg.Port) != "",
			"smtp_username_set": strings.TrimSpace(cfg.Username) != "",
			"smtp_password_set": strings.TrimSpace(cfg.Password) != "",
			"smtp_from_set":     strings.TrimSpace(cfg.From) != "",
			"to":                strings.TrimSpace(to),
			"subject":           subject,
		})
		return errors.New("email service is disabled or not configured")
	}

	if strings.TrimSpace(cfg.Host) == "" || strings.TrimSpace(cfg.Username) == "" || strings.TrimSpace(cfg.Password) == "" {
		logger.Warn("Email service disabled: missing SMTP configuration", map[string]interface{}{
			"smtp_host_set":     strings.TrimSpace(cfg.Host) != "",
			"smtp_port_set":     strings.TrimSpace(cfg.Port) != "",
			"smtp_username_set": strings.TrimSpace(cfg.Username) != "",
			"smtp_password_set": strings.TrimSpace(cfg.Password) != "",
			"smtp_from_set":     strings.TrimSpace(cfg.From) != "",
			"to":                strings.TrimSpace(to),
			"subject":           subject,
		})
		return errors.New("email service is disabled or not configured")
	}

	start := time.Now()

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	var resolvedIPs []string
	var dnsLookupErr error
	if ips, err := net.LookupIP(cfg.Host); err == nil {
		for i, ip := range ips {
			if i >= 3 {
				break
			}
			resolvedIPs = append(resolvedIPs, ip.String())
		}
	} else {
		dnsLookupErr = err
		resolvedIPs = []string{fmt.Sprintf("lookup_error:%s", err.Error())}
		logger.Warn("SMTP DNS lookup failed", map[string]interface{}{
			"smtp_host": cfg.Host,
			"error":     err.Error(),
		})
	}

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

	dialTimeout := 12 * time.Second
	messageBytes := []byte(builder.String())
	messageSize := len(messageBytes)
	startFields := map[string]interface{}{
		"smtp_host":              cfg.Host,
		"smtp_port":              cfg.Port,
		"smtp_from":              cfg.From,
		"smtp_username":          cfg.Username,
		"smtp_username_set":      cfg.Username != "",
		"smtp_username_length":   len(strings.TrimSpace(cfg.Username)),
		"smtp_password_set":      cfg.Password != "",
		"smtp_password_length":   len(strings.TrimSpace(cfg.Password)),
		"smtp_resolved_ips":      strings.Join(resolvedIPs, ","),
		"smtp_lookup_error":      "",
		"smtp_address":           addr,
		"to":                     strings.TrimSpace(to),
		"subject":                subject,
		"message_bytes":          messageSize,
		"auth_mechanism":         "PLAIN",
		"dial_timeout_ms":        dialTimeout.Milliseconds(),
		"from_matches_username":  strings.EqualFold(strings.TrimSpace(cfg.From), strings.TrimSpace(cfg.Username)),
		"password_matches_empty": strings.TrimSpace(cfg.Password) == "",
	}
	if dnsLookupErr != nil {
		startFields["smtp_lookup_error"] = dnsLookupErr.Error()
	}
	logger.Info("Starting SMTP send", startFields)

	dialer := net.Dialer{Timeout: dialTimeout}
	dialStart := time.Now()
	conn, dialErr := dialer.Dial("tcp", addr)
	dialDuration := time.Since(dialStart)
	if dialErr != nil {
		fields := map[string]interface{}{
			"smtp_host":              cfg.Host,
			"smtp_port":              cfg.Port,
			"smtp_username":          cfg.Username,
			"smtp_username_set":      cfg.Username != "",
			"smtp_username_length":   len(strings.TrimSpace(cfg.Username)),
			"smtp_password_set":      cfg.Password != "",
			"smtp_password_length":   len(strings.TrimSpace(cfg.Password)),
			"smtp_resolved_ips":      strings.Join(resolvedIPs, ","),
			"smtp_lookup_error":      startFields["smtp_lookup_error"],
			"dial_timeout_ms":        dialTimeout.Milliseconds(),
			"dial_duration_ms":       dialDuration.Milliseconds(),
			"to":                     strings.TrimSpace(to),
			"subject":                subject,
			"message_bytes":          messageSize,
			"smtp_address":           addr,
			"auth_mechanism":         "PLAIN",
			"from_matches_username":  strings.EqualFold(strings.TrimSpace(cfg.From), strings.TrimSpace(cfg.Username)),
			"password_matches_empty": strings.TrimSpace(cfg.Password) == "",
		}
		logger.Error(dialErr, "SMTP dial failed", fields)
		return fmt.Errorf("failed to dial SMTP server: %w", dialErr)
	}
	_ = conn.Close()
	logger.Info("SMTP dial succeeded", map[string]interface{}{
		"smtp_host":         cfg.Host,
		"smtp_port":         cfg.Port,
		"smtp_resolved_ips": strings.Join(resolvedIPs, ","),
		"smtp_lookup_error": startFields["smtp_lookup_error"],
		"dial_timeout_ms":   dialTimeout.Milliseconds(),
		"dial_duration_ms":  dialDuration.Milliseconds(),
		"message_bytes":     messageSize,
		"smtp_address":      addr,
		"auth_mechanism":    "PLAIN",
	})

	err := smtp.SendMail(addr, auth, cfg.From, []string{to}, messageBytes)
	duration := time.Since(start)

	fields := map[string]interface{}{
		"smtp_host":              cfg.Host,
		"smtp_port":              cfg.Port,
		"smtp_from":              cfg.From,
		"smtp_username":          cfg.Username,
		"smtp_username_set":      cfg.Username != "",
		"smtp_username_length":   len(strings.TrimSpace(cfg.Username)),
		"smtp_password_set":      cfg.Password != "",
		"smtp_password_length":   len(strings.TrimSpace(cfg.Password)),
		"smtp_resolved_ips":      strings.Join(resolvedIPs, ","),
		"smtp_lookup_error":      startFields["smtp_lookup_error"],
		"smtp_address":           addr,
		"duration_ms":            duration.Milliseconds(),
		"dial_timeout_ms":        dialTimeout.Milliseconds(),
		"dial_duration_ms":       dialDuration.Milliseconds(),
		"to":                     strings.TrimSpace(to),
		"subject":                subject,
		"message_bytes":          messageSize,
		"auth_mechanism":         "PLAIN",
		"from_matches_username":  strings.EqualFold(strings.TrimSpace(cfg.From), strings.TrimSpace(cfg.Username)),
		"password_matches_empty": strings.TrimSpace(cfg.Password) == "",
	}

	if err != nil {
		logger.Error(err, "Failed to send email via SMTP", fields)
		return err
	}

	logger.Info("Email sent via SMTP", fields)
	return nil
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
