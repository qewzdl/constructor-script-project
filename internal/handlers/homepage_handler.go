package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HomepageHandler struct {
	homepageService *service.HomepageService
}

func NewHomepageHandler(homepageService *service.HomepageService) *HomepageHandler {
	return &HomepageHandler{homepageService: homepageService}
}

func (h *HomepageHandler) Get(c *gin.Context) {
	if h == nil || h.homepageService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "homepage service unavailable"})
		return
	}

	selection, err := h.homepageService.GetSelection()
	if err != nil {
		logger.Error(err, "Failed to load homepage selection", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load homepage selection"})
		return
	}

	options, err := h.homepageService.ListOptions()
	if err != nil {
		logger.Error(err, "Failed to load pages for homepage selection", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load pages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"homepage": selection,
		"options":  options,
	})
}

func (h *HomepageHandler) Update(c *gin.Context) {
	if h == nil || h.homepageService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "homepage service unavailable"})
		return
	}

	var req models.UpdateHomepageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var selection *models.HomepagePage
	message := "Homepage updated successfully."

	if req.PageID == nil || *req.PageID == 0 {
		if err := h.homepageService.ClearHomepage(); err != nil {
			logger.Error(err, "Failed to clear homepage selection", nil)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update homepage"})
			return
		}
		message = "Homepage selection cleared. The site will use the page assigned to \"/\"."
	} else {
		pageID := *req.PageID
		result, err := h.homepageService.SetHomepage(pageID)
		if err != nil {
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				c.JSON(http.StatusNotFound, gin.H{"error": "Page not found"})
			case errors.Is(err, service.ErrHomepagePageNotPublished), errors.Is(err, service.ErrHomepagePageScheduled):
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			default:
				logger.Error(err, "Failed to update homepage selection", map[string]interface{}{"page_id": pageID})
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update homepage"})
			}
			return
		}
		selection = result
		if selection != nil {
			message = fmt.Sprintf("\"%s\" set as the homepage.", selection.Title)
		}
	}

	if selection == nil {
		current, err := h.homepageService.GetSelection()
		if err != nil {
			logger.Error(err, "Failed to load homepage selection", nil)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load homepage selection"})
			return
		}
		selection = current
	}

	options, err := h.homepageService.ListOptions()
	if err != nil {
		logger.Error(err, "Failed to load pages for homepage selection", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load pages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  message,
		"homepage": selection,
		"options":  options,
	})
}
