package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/lang"
	"constructor-script-backend/pkg/logger"
	blogservice "constructor-script-backend/plugins/blog/service"

	"github.com/gin-gonic/gin"
)

type SetupHandler struct {
	setupService *service.SetupService
	fontService  *service.FontService
	config       *config.Config
}

func NewSetupHandler(setupService *service.SetupService, fontService *service.FontService, cfg *config.Config) *SetupHandler {
	return &SetupHandler{
		setupService: setupService,
		fontService:  fontService,
		config:       cfg,
	}
}

func (h *SetupHandler) Status(c *gin.Context) {
	if h.setupService == nil {
		defaults := h.defaultSiteSettings()
		h.sanitizeSensitiveSettings(&defaults)
		c.JSON(http.StatusOK, gin.H{
			"setup_required": false,
			"site":           defaults,
		})
		return
	}

	complete, err := h.setupService.IsSetupComplete()
	if err != nil {
		logger.Error(err, "Failed to determine setup status", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check setup status"})
		return
	}

	settings, err := h.setupService.GetSiteSettings(h.defaultSiteSettings())
	if err != nil {
		logger.Error(err, "Failed to load site settings", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load site settings"})
		return
	}

	h.applyFontSettings(&settings)
	h.sanitizeSensitiveSettings(&settings)

	response := models.SetupStatusResponse{
		SetupRequired: !complete,
		Site:          settings,
	}

	// If setup is required, get the current progress
	if !complete {
		progress, err := h.setupService.GetSetupProgress()
		if err != nil {
			logger.Error(err, "Failed to get setup progress", nil)
		} else {
			response.CurrentStep = progress.CurrentStep
			response.Progress = progress
			// Don't expose sensitive data
			response.Progress.Admin.Password = ""
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetStepProgress returns the current setup step and progress
func (h *SetupHandler) GetStepProgress(c *gin.Context) {
	if h.setupService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setup service not available"})
		return
	}

	progress, err := h.setupService.GetSetupProgress()
	if err != nil {
		logger.Error(err, "Failed to get setup progress", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get setup progress"})
		return
	}

	// Don't expose sensitive data
	progress.Admin.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"progress": progress,
	})
}

// SaveStep saves data for a specific setup step
func (h *SetupHandler) SaveStep(c *gin.Context) {
	if h.setupService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setup service not available"})
		return
	}

	var req models.SetupStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleValidationError(c, err)
		return
	}

	progress, err := h.setupService.SaveStepData(req)
	if err != nil {
		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		logger.Error(err, "Failed to save step data", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save step data"})
		return
	}

	// Don't expose sensitive data
	progress.Admin.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"message":  "Step data saved successfully",
		"progress": progress,
	})
}

// CompleteStepwiseSetup finalizes the setup after all steps are completed
func (h *SetupHandler) CompleteStepwiseSetup(c *gin.Context) {
	if h.setupService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setup service not available"})
		return
	}

	defaults := h.defaultSiteSettings()

	_, err := h.setupService.CompleteStepwiseSetup(defaults)
	if err != nil {
		if errors.Is(err, service.ErrSetupAlreadyCompleted) {
			c.JSON(http.StatusConflict, gin.H{"error": "Setup has already been completed"})
			return
		}

		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		logger.Error(err, "Failed to complete setup", map[string]interface{}{
			"error_message": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to complete setup",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Setup completed successfully"})
}

func (h *SetupHandler) Complete(c *gin.Context) {
	if h.setupService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setup service not available"})
		return
	}

	var req models.SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleValidationError(c, err)
		return
	}

	// Validate request data
	if err := h.validateSetupRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	defaults := h.defaultSiteSettings()

	if strings.TrimSpace(req.SiteDefaultLanguage) == "" {
		req.SiteDefaultLanguage = defaults.DefaultLanguage
	}
	if len(req.SiteSupportedLanguages) == 0 {
		req.SiteSupportedLanguages = defaults.SupportedLanguages
	}

	_, err := h.setupService.CompleteSetup(req, defaults)
	if err != nil {
		if errors.Is(err, service.ErrSetupAlreadyCompleted) {
			c.JSON(http.StatusConflict, gin.H{"error": "Setup has already been completed"})
			return
		}

		var validationErr *service.ValidationError
		if errors.As(err, &validationErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		logger.Error(err, "Failed to complete setup", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete setup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Setup completed successfully"})
}

func (h *SetupHandler) defaultSiteSettings() models.SiteSettings {
	var logo string
	var contactEmail string
	if h.config != nil {
		logo = h.config.SiteLogo
		contactEmail = strings.TrimSpace(h.config.SMTPFrom)
	}
	if strings.TrimSpace(logo) == "" {
		logo = "/static/icons/logo.svg"
	}

	defaultLanguage := lang.Default
	supportedLanguages := []string{defaultLanguage}
	if h.config != nil {
		defaultLanguage = h.config.DefaultLanguage
		if len(h.config.SupportedLanguages) > 0 {
			supportedLanguages = append([]string(nil), h.config.SupportedLanguages...)
		} else {
			supportedLanguages = []string{defaultLanguage}
		}
	}

	settings := models.SiteSettings{
		Logo:                     logo,
		ContactEmail:             contactEmail,
		FooterText:               "",
		UnusedTagRetentionHours:  blogservice.DefaultUnusedTagRetentionHours,
		DefaultLanguage:          defaultLanguage,
		SupportedLanguages:       supportedLanguages,
		StripeSecretKey:          "",
		StripePublishableKey:     "",
		StripeWebhookSecret:      "",
		CourseCheckoutSuccessURL: "",
		CourseCheckoutCancelURL:  "",
		CourseCheckoutCurrency:   "",
		Subtitles:                models.SubtitleSettings{},
	}

	if h.config != nil {
		settings.Name = h.config.SiteName
		settings.Description = h.config.SiteDescription
		settings.URL = h.config.SiteURL
		settings.Favicon = h.config.SiteFavicon
		settings.FaviconType = models.DetectFaviconType(h.config.SiteFavicon)
		settings.StripeSecretKey = strings.TrimSpace(h.config.StripeSecretKey)
		settings.StripePublishableKey = strings.TrimSpace(h.config.StripePublishableKey)
		settings.StripeWebhookSecret = strings.TrimSpace(h.config.StripeWebhookSecret)
		settings.CourseCheckoutSuccessURL = strings.TrimSpace(h.config.CourseCheckoutSuccessURL)
		settings.CourseCheckoutCancelURL = strings.TrimSpace(h.config.CourseCheckoutCancelURL)
		settings.CourseCheckoutCurrency = strings.ToLower(strings.TrimSpace(h.config.CourseCheckoutCurrency))

		settings.Subtitles.Enabled = h.config.SubtitleGenerationEnabled
		settings.Subtitles.Provider = strings.TrimSpace(h.config.SubtitleProvider)
		settings.Subtitles.PreferredName = strings.TrimSpace(h.config.SubtitlePreferredName)
		settings.Subtitles.Language = strings.TrimSpace(h.config.SubtitleLanguage)
		settings.Subtitles.Prompt = strings.TrimSpace(h.config.SubtitlePrompt)
		settings.Subtitles.OpenAIModel = strings.TrimSpace(h.config.OpenAIModel)
		settings.Subtitles.OpenAIAPIKey = strings.TrimSpace(h.config.OpenAIAPIKey)
		if h.config.SubtitleTemperature != nil {
			value := *h.config.SubtitleTemperature
			settings.Subtitles.Temperature = &value
		}
	}

	h.applyFontSettings(&settings)

	return settings
}

func (h *SetupHandler) defaultEmailSettings() models.EmailSettings {
	if h.config == nil {
		return models.EmailSettings{
			Port:        "587",
			EnableEmail: true,
		}
	}

	port := strings.TrimSpace(h.config.SMTPPort)
	if port == "" {
		port = "587"
	}
	password := strings.TrimSpace(h.config.SMTPPassword)
	from := strings.TrimSpace(h.config.SMTPFrom)
	contact := from

	return models.EmailSettings{
		Host:         strings.TrimSpace(h.config.SMTPHost),
		Port:         port,
		Username:     strings.TrimSpace(h.config.SMTPUsername),
		From:         from,
		ContactEmail: contact,
		PasswordSet:  password != "",
		EnableEmail:  true,
	}
}

func (h *SetupHandler) applyFontSettings(settings *models.SiteSettings) {
	if settings == nil {
		return
	}

	fonts := []models.FontAsset{}
	if h.fontService != nil {
		if list, err := h.fontService.List(); err != nil {
			logger.Error(err, "Failed to load fonts", nil)
			fonts = service.DefaultFontAssets()
		} else {
			fonts = list
		}
	} else {
		fonts = service.DefaultFontAssets()
	}

	settings.Fonts = fonts
	settings.FontPreconnects = service.CollectFontPreconnects(fonts)
}

func (h *SetupHandler) sanitizeSensitiveSettings(settings *models.SiteSettings) {
	if settings == nil {
		return
	}
	settings.StripeSecretKey = maskIfSet(settings.StripeSecretKey)
	settings.StripeWebhookSecret = maskIfSet(settings.StripeWebhookSecret)
	settings.Subtitles.OpenAIAPIKey = maskIfSet(settings.Subtitles.OpenAIAPIKey)
}

func maskIfSet(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return "********"
}

func (h *SetupHandler) GetSiteSettings(c *gin.Context) {
	defaults := h.defaultSiteSettings()

	if h.setupService == nil {
		c.JSON(http.StatusOK, gin.H{"site": defaults})
		return
	}

	settings, err := h.setupService.GetSiteSettings(defaults)
	if err != nil {
		logger.Error(err, "Failed to load site settings", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load site settings"})
		return
	}

	h.applyFontSettings(&settings)

	c.JSON(http.StatusOK, gin.H{"site": settings})
}

func (h *SetupHandler) GetEmailSettings(c *gin.Context) {
	defaults := h.defaultEmailSettings()

	if h.setupService == nil {
		c.JSON(http.StatusOK, gin.H{"email": defaults})
		return
	}

	settings, err := h.setupService.GetEmailSettings(defaults)
	if err != nil {
		logger.Error(err, "Failed to load email settings", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load email settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"email": settings})
}

func (h *SetupHandler) UpdateSiteSettings(c *gin.Context) {
	if h.setupService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setup service not available"})
		return
	}

	var req models.UpdateSiteSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	defaults := h.defaultSiteSettings()

	if strings.TrimSpace(req.DefaultLanguage) == "" {
		req.DefaultLanguage = defaults.DefaultLanguage
	}
	if len(req.SupportedLanguages) == 0 {
		req.SupportedLanguages = defaults.SupportedLanguages
	}

	if err := h.setupService.UpdateSiteSettings(req, defaults); err != nil {
		var validationErr *service.ValidationError
		if errors.As(err, &validationErr) {
			fields := map[string]interface{}{
				"field": validationErr.Field,
				"error": validationErr.Message,
			}
			if prefix := safeKeyPrefix(req, validationErr.Field); prefix != "" {
				fields["key_prefix"] = prefix
				fields["key_length"] = safeKeyLength(req, validationErr.Field)
			}

			logger.Warn("Site settings validation failed", fields)
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		logger.Error(err, "Failed to update site settings", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update site settings"})
		return
	}

	settings, err := h.setupService.GetSiteSettings(defaults)
	if err != nil {
		logger.Error(err, "Failed to load site settings", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load site settings"})
		return
	}

	h.applyFontSettings(&settings)

	c.JSON(http.StatusOK, gin.H{"message": "Site settings updated", "site": settings})
}

func (h *SetupHandler) UpdateEmailSettings(c *gin.Context) {
	if h.setupService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setup service not available"})
		return
	}

	var req models.UpdateEmailSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	defaults := h.defaultEmailSettings()

	logger.Info("Email settings update requested", map[string]interface{}{
		"host":             strings.TrimSpace(req.Host),
		"port":             strings.TrimSpace(req.Port),
		"from":             strings.TrimSpace(req.From),
		"username_set":     strings.TrimSpace(req.Username) != "",
		"contact_email":    strings.TrimSpace(req.ContactEmail),
		"password_provided": strings.TrimSpace(req.Password) != "",
	})

	if err := h.setupService.UpdateEmailSettings(req, defaults); err != nil {
		var validationErr *service.ValidationError
		if errors.As(err, &validationErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		logger.Error(err, "Failed to update email settings", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update email settings"})
		return
	}

	settings, err := h.setupService.GetEmailSettings(defaults)
	if err != nil {
		logger.Error(err, "Failed to load email settings", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load email settings"})
		return
	}

	logger.Info("Email settings updated", map[string]interface{}{
		"host":          strings.TrimSpace(settings.Host),
		"port":          strings.TrimSpace(settings.Port),
		"from":          strings.TrimSpace(settings.From),
		"username_set":  strings.TrimSpace(settings.Username) != "",
		"contact_email": strings.TrimSpace(settings.ContactEmail),
		"password_set":  settings.PasswordSet,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Email settings updated", "email": settings})
}

func safeKeyPrefix(req models.UpdateSiteSettingsRequest, field string) string {
	value := strings.TrimSpace(selectKeyByField(req, field))
	if value == "" {
		return ""
	}

	if len(value) <= 6 {
		return value
	}

	return value[:6] + "..."
}

func safeKeyLength(req models.UpdateSiteSettingsRequest, field string) int {
	value := strings.TrimSpace(selectKeyByField(req, field))
	return len(value)
}

func selectKeyByField(req models.UpdateSiteSettingsRequest, field string) string {
	switch field {
	case "stripe_secret_key":
		return req.StripeSecretKey
	case "stripe_publishable_key":
		return req.StripePublishableKey
	case "stripe_webhook_secret":
		return req.StripeWebhookSecret
	default:
		return ""
	}
}

func (h *SetupHandler) UploadFavicon(c *gin.Context) {
	if h.setupService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setup service not available"})
		return
	}

	file, err := c.FormFile("favicon")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No favicon uploaded"})
		return
	}

	url, faviconType, replaceErr := h.setupService.ReplaceFavicon(file)
	if replaceErr != nil {
		var invalidErr *service.InvalidFaviconError
		if errors.As(replaceErr, &invalidErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": invalidErr.Error()})
			return
		}

		logger.Error(replaceErr, "Failed to upload favicon", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload favicon"})
		return
	}

	settings, err := h.setupService.GetSiteSettings(h.defaultSiteSettings())
	if err != nil {
		logger.Error(err, "Failed to load site settings", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load site settings"})
		return
	}

	h.applyFontSettings(&settings)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Favicon updated successfully",
		"favicon":      url,
		"favicon_type": faviconType,
		"site":         settings,
	})
}

func (h *SetupHandler) UploadLogo(c *gin.Context) {
	if h.setupService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setup service not available"})
		return
	}

	file, err := c.FormFile("logo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No logo uploaded"})
		return
	}

	url, replaceErr := h.setupService.ReplaceLogo(file)
	if replaceErr != nil {
		var invalidErr *service.InvalidLogoError
		if errors.As(replaceErr, &invalidErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": invalidErr.Error()})
			return
		}

		logger.Error(replaceErr, "Failed to upload logo", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload logo"})
		return
	}

	settings, err := h.setupService.GetSiteSettings(h.defaultSiteSettings())
	if err != nil {
		logger.Error(err, "Failed to load site settings", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load site settings"})
		return
	}

	h.applyFontSettings(&settings)

	c.JSON(http.StatusOK, gin.H{
		"message": "Logo updated successfully",
		"logo":    url,
		"site":    settings,
	})
}

// validateSetupRequest validates the setup request data
func (h *SetupHandler) validateSetupRequest(req *models.SetupRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}

	// Validate username
	if len(strings.TrimSpace(req.AdminUsername)) < 3 {
		return errors.New("username must be at least 3 characters")
	}

	// Validate email format
	if !strings.Contains(req.AdminEmail, "@") {
		return errors.New("invalid email address")
	}

	// Validate password strength
	if len(req.AdminPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	// Validate site URL
	if req.SiteURL != "" {
		if _, err := url.Parse(req.SiteURL); err != nil {
			return errors.New("invalid site URL")
		}
	}

	return nil
}

// handleValidationError processes validation errors and returns user-friendly messages
func (h *SetupHandler) handleValidationError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// Extract field-specific errors from gin binding errors
	message := "Validation failed"
	if err.Error() != "" {
		message = err.Error()
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"error": message,
	})
}
