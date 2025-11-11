package handlers

import (
	"errors"
	"net/http"
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

	c.JSON(http.StatusOK, gin.H{
		"setup_required": !complete,
		"site":           settings,
	})
}

func (h *SetupHandler) Complete(c *gin.Context) {
	if h.setupService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setup service not available"})
		return
	}

	var req models.SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Setup has already been completed"})
			return
		}

		logger.Error(err, "Failed to complete setup", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete setup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Setup completed"})
}

func (h *SetupHandler) defaultSiteSettings() models.SiteSettings {
	var logo string
	if h.config != nil {
		logo = h.config.SiteLogo
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
	settings.StripeSecretKey = ""
	settings.StripeWebhookSecret = ""
	settings.Subtitles.OpenAIAPIKey = ""
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
		logger.Error(err, "Failed to update site settings", nil)
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
