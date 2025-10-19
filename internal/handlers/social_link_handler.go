package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SocialLinkHandler struct {
	service *service.SocialLinkService
}

func NewSocialLinkHandler(service *service.SocialLinkService) *SocialLinkHandler {
	return &SocialLinkHandler{service: service}
}

func (h *SocialLinkHandler) List(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not configured"})
		return
	}

	links, err := h.service.List()
	if err != nil {
		logger.Error(err, "Failed to load social links", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load social links"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"social_links": links})
}

func (h *SocialLinkHandler) Create(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not configured"})
		return
	}

	var req models.CreateSocialLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	link, err := h.service.Create(req)
	if err != nil {
		logger.Error(err, "Failed to create social link", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"social_link": link})
}

func (h *SocialLinkHandler) Update(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not configured"})
		return
	}

	idParam := c.Param("id")
	idValue, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid social link ID"})
		return
	}

	var req models.UpdateSocialLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	link, err := h.service.Update(uint(idValue), req)
	if err != nil {
		logger.Error(err, "Failed to update social link", map[string]interface{}{"id": idValue})
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"social_link": link})
}

func (h *SocialLinkHandler) Delete(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not configured"})
		return
	}

	idParam := c.Param("id")
	idValue, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid social link ID"})
		return
	}

	if err := h.service.Delete(uint(idValue)); err != nil {
		logger.Error(err, "Failed to delete social link", map[string]interface{}{"id": idValue})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete social link"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Social link deleted"})
}
