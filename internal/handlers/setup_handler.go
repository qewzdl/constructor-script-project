package handlers

import (
	"errors"
	"net/http"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"

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
		return models.SiteSettings{Logo: logo}
	}

	return models.SiteSettings{
		Name:        h.config.SiteName,
		Description: h.config.SiteDescription,
		URL:         h.config.SiteURL,
		Favicon:     h.config.SiteFavicon,
		FaviconType: models.DetectFaviconType(h.config.SiteFavicon),
		Logo:        logo,
	}
}
