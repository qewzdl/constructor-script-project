package handlers

import (
	"errors"
	"net/http"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"
	postservice "constructor-script-backend/plugins/posts/service"

	"github.com/gin-gonic/gin"
)

type SetupHandler struct {
	setupService *service.SetupService
	config       *config.Config
}

func NewSetupHandler(setupService *service.SetupService, cfg *config.Config) *SetupHandler {
	return &SetupHandler{
		setupService: setupService,
		config:       cfg,
	}
}

func (h *SetupHandler) Status(c *gin.Context) {
	if h.setupService == nil {
		c.JSON(http.StatusOK, gin.H{
			"setup_required": false,
			"site":           h.defaultSiteSettings(),
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

	_, err := h.setupService.CompleteSetup(req)
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
	logo := "/static/icons/logo.svg"
	if h.config == nil {
		return models.SiteSettings{Logo: logo, UnusedTagRetentionHours: postservice.DefaultUnusedTagRetentionHours}
	}

	return models.SiteSettings{
		Name:                    h.config.SiteName,
		Description:             h.config.SiteDescription,
		URL:                     h.config.SiteURL,
		Favicon:                 h.config.SiteFavicon,
		FaviconType:             models.DetectFaviconType(h.config.SiteFavicon),
		Logo:                    logo,
		UnusedTagRetentionHours: postservice.DefaultUnusedTagRetentionHours,
	}
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

	if err := h.setupService.UpdateSiteSettings(req); err != nil {
		logger.Error(err, "Failed to update site settings", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update site settings"})
		return
	}

	settings, err := h.setupService.GetSiteSettings(h.defaultSiteSettings())
	if err != nil {
		logger.Error(err, "Failed to load site settings", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load site settings"})
		return
	}

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

	c.JSON(http.StatusOK, gin.H{
		"message": "Logo updated successfully",
		"logo":    url,
		"site":    settings,
	})
}
