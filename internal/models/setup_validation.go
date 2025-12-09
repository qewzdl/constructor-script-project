package models

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"constructor-script-backend/pkg/lang"
)

var (
	emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// SetupStepValidator interface for step validation
type SetupStepValidator interface {
	Validate() error
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// Validate validates SiteInfoData
func (d *SiteInfoData) Validate() error {
	name := strings.TrimSpace(d.Name)
	if name == "" {
		return NewValidationError("site_name", "is required")
	}
	if len(name) > 255 {
		return NewValidationError("site_name", "must not exceed 255 characters")
	}

	siteURL := strings.TrimSpace(d.URL)
	if siteURL == "" {
		return NewValidationError("site_url", "is required")
	}

	parsedURL, err := url.Parse(siteURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return NewValidationError("site_url", "must be a valid HTTP or HTTPS URL")
	}

	if len(d.Description) > 1000 {
		return NewValidationError("site_description", "must not exceed 1000 characters")
	}

	return nil
}

// Validate validates AdminData
func (d *AdminData) Validate() error {
	username := strings.TrimSpace(d.Username)
	if len(username) < 3 {
		return NewValidationError("admin_username", "must be at least 3 characters")
	}
	if len(username) > 50 {
		return NewValidationError("admin_username", "must not exceed 50 characters")
	}

	email := strings.TrimSpace(d.Email)
	if email == "" || !emailPattern.MatchString(email) {
		return NewValidationError("admin_email", "must be a valid email address")
	}

	// Password validation (before hashing)
	if d.Password != "" && len(d.Password) < 8 {
		return NewValidationError("admin_password", "must be at least 8 characters")
	}
	if len(d.Password) > 128 {
		return NewValidationError("admin_password", "must not exceed 128 characters")
	}

	return nil
}

// Validate validates LanguagesData
func (d *LanguagesData) Validate() error {
	if d.DefaultLanguage != "" {
		if _, err := lang.Normalize(d.DefaultLanguage); err != nil {
			return NewValidationError("default_language", "is not a valid language code")
		}
	}

	if d.SupportedLanguages != "" {
		langs := strings.Split(d.SupportedLanguages, ",")
		for _, langCode := range langs {
			trimmed := strings.TrimSpace(langCode)
			if trimmed == "" {
				continue
			}
			if _, err := lang.Normalize(trimmed); err != nil {
				return NewValidationError("supported_languages", fmt.Sprintf("invalid language code: %s", trimmed))
			}
		}
	}

	return nil
}

// ValidateStep validates a setup step request
func (r *SetupStepRequest) ValidateStep() error {
	step := SetupStep(r.Step)

	if !step.IsValid() {
		return NewValidationError("step", "invalid setup step")
	}

	switch step {
	case SetupStepSiteInfo:
		data := r.ToSiteInfoData()
		return data.Validate()

	case SetupStepAdmin:
		data := r.ToAdminData()
		return data.Validate()

	case SetupStepLanguages:
		data := r.ToLanguagesData()
		return data.Validate()

	default:
		return NewValidationError("step", "unknown setup step")
	}
}
