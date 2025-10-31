package handlers

import (
	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	blogservice "constructor-script-backend/plugins/blog/service"
)

func ResolveSiteSettings(cfg *config.Config, setupService *service.SetupService) (models.SiteSettings, error) {
	defaults := models.SiteSettings{
		Name:                    cfg.SiteName,
		Description:             cfg.SiteDescription,
		URL:                     cfg.SiteURL,
		Favicon:                 cfg.SiteFavicon,
		FaviconType:             models.DetectFaviconType(cfg.SiteFavicon),
		Logo:                    "/static/icons/logo.svg",
		UnusedTagRetentionHours: blogservice.DefaultUnusedTagRetentionHours,
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

	return settings, nil
}
