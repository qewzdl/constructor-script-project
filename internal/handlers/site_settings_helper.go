package handlers

import (
	"strings"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	blogservice "constructor-script-backend/plugins/blog/service"
	languageservice "constructor-script-backend/plugins/language/service"
)

func ResolveSiteSettings(cfg *config.Config, setupService *service.SetupService, languageService *languageservice.LanguageService) (models.SiteSettings, error) {
	defaultLanguage := cfg.DefaultLanguage
	supportedLanguages := append([]string(nil), cfg.SupportedLanguages...)
	if languageService != nil {
		if fallbackDefault, fallbackSupported := languageService.Defaults(); fallbackDefault != "" {
			defaultLanguage = fallbackDefault
			supportedLanguages = append([]string(nil), fallbackSupported...)
		}
	}

	var subtitleTemp *float32
	if cfg.SubtitleTemperature != nil {
		value := *cfg.SubtitleTemperature
		subtitleTemp = &value
	}

	defaults := models.SiteSettings{
		Name:                     cfg.SiteName,
		Description:              cfg.SiteDescription,
		URL:                      cfg.SiteURL,
		Favicon:                  cfg.SiteFavicon,
		FaviconType:              models.DetectFaviconType(cfg.SiteFavicon),
		Logo:                     cfg.SiteLogo,
		ContactEmail:             strings.TrimSpace(cfg.SMTPFrom),
		FooterText:               strings.TrimSpace(cfg.SiteDescription),
		UnusedTagRetentionHours:  blogservice.DefaultUnusedTagRetentionHours,
		DefaultLanguage:          defaultLanguage,
		SupportedLanguages:       supportedLanguages,
		StripeSecretKey:          cfg.StripeSecretKey,
		StripePublishableKey:     cfg.StripePublishableKey,
		StripeWebhookSecret:      cfg.StripeWebhookSecret,
		CourseCheckoutSuccessURL: cfg.CourseCheckoutSuccessURL,
		CourseCheckoutCancelURL:  cfg.CourseCheckoutCancelURL,
		CourseCheckoutCurrency:   strings.ToLower(strings.TrimSpace(cfg.CourseCheckoutCurrency)),
		Subtitles: models.SubtitleSettings{
			Enabled:       cfg.SubtitleGenerationEnabled,
			Provider:      strings.TrimSpace(cfg.SubtitleProvider),
			PreferredName: strings.TrimSpace(cfg.SubtitlePreferredName),
			Language:      strings.TrimSpace(cfg.SubtitleLanguage),
			Prompt:        strings.TrimSpace(cfg.SubtitlePrompt),
			Temperature:   subtitleTemp,
			OpenAIModel:   strings.TrimSpace(cfg.OpenAIModel),
			OpenAIAPIKey:  strings.TrimSpace(cfg.OpenAIAPIKey),
		},
	}

	if strings.TrimSpace(defaults.Logo) == "" {
		defaults.Logo = "/static/icons/logo.svg"
	}

	if len(defaults.SupportedLanguages) == 0 {
		defaults.SupportedLanguages = []string{defaults.DefaultLanguage}
	}

	if setupService == nil {
		return defaults, nil
	}

	settings, err := setupService.GetSiteSettings(defaults)
	if err != nil {
		return defaults, err
	}

	if settings.FaviconType == "" {
		settings.FaviconType = models.DetectFaviconType(settings.Favicon)
	}

	if len(settings.SupportedLanguages) == 0 {
		settings.SupportedLanguages = []string{settings.DefaultLanguage}
	}

	settings.StripeSecretKey = ""
	settings.StripeWebhookSecret = ""
	settings.Subtitles.OpenAIAPIKey = ""

	return settings, nil
}
