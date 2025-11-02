package handlers

import (
	"errors"
	"net/http"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// FontHandler provides HTTP handlers for managing external font resources.
type FontHandler struct {
	service *service.FontService
}

// NewFontHandler constructs a font handler instance.
func NewFontHandler(service *service.FontService) *FontHandler {
	return &FontHandler{service: service}
}

// List returns all configured font assets.
func (h *FontHandler) List(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not configured"})
		return
	}

	fonts, err := h.service.List()
	if err != nil {
		logger.Error(err, "Failed to load fonts", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load fonts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"fonts": fonts})
}

// Create adds a new font asset definition.
func (h *FontHandler) Create(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not configured"})
		return
	}

	var req models.CreateFontAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	font, err := h.service.Create(req)
	if err != nil {
		logger.Error(err, "Failed to create font", nil)
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrInvalidFontSnippet):
			status = http.StatusBadRequest
		case errors.Is(err, service.ErrFontNotFound):
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"font": font})
}

// Update modifies an existing font asset.
func (h *FontHandler) Update(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not configured"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid font ID"})
		return
	}

	var req models.UpdateFontAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	font, err := h.service.Update(id, req)
	if err != nil {
		logger.Error(err, "Failed to update font", map[string]interface{}{"id": id})
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrFontNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"font": font})
}

// Delete removes a font asset.
func (h *FontHandler) Delete(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not configured"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid font ID"})
		return
	}

	if err := h.service.Delete(id); err != nil {
		logger.Error(err, "Failed to delete font", map[string]interface{}{"id": id})
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrFontNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Font deleted"})
}

// Reorder updates the ordering for all fonts.
func (h *FontHandler) Reorder(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not configured"})
		return
	}

	var req models.ReorderFontAssetsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No font order provided"})
		return
	}

	if err := h.service.Reorder(req.Items); err != nil {
		logger.Error(err, "Failed to reorder fonts", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder fonts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Font order updated"})
}
