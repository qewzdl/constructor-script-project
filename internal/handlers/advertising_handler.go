package handlers

import (
	"errors"
	"net/http"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

type AdvertisingHandler struct {
	service *service.AdvertisingService
}

func NewAdvertisingHandler(svc *service.AdvertisingService) *AdvertisingHandler {
	return &AdvertisingHandler{service: svc}
}

func (h *AdvertisingHandler) Get(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Advertising service not available"})
		return
	}

	settings, err := h.service.GetSettings()
	if err != nil {
		logger.Error(err, "Failed to load advertising settings", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load advertising settings"})
		return
	}

	providers := h.service.Providers()

	c.JSON(http.StatusOK, gin.H{
		"settings":  settings,
		"providers": providers,
	})
}

func (h *AdvertisingHandler) Update(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Advertising service not available"})
		return
	}

	var req models.UpdateAdvertisingSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings, err := h.service.UpdateSettings(req)
	if err != nil {
		var validationErr *service.AdvertisingValidationError
		if errors.As(err, &validationErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		logger.Error(err, "Failed to update advertising settings", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update advertising settings"})
		return
	}

	providers := h.service.Providers()

	c.JSON(http.StatusOK, gin.H{
		"message":   "Advertising settings updated",
		"settings":  settings,
		"providers": providers,
	})
}
