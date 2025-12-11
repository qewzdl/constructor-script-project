package handlers

import (
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type PageHandler struct {
	pageService *service.PageService
}

func NewPageHandler(pageService *service.PageService) *PageHandler {
	return &PageHandler{pageService: pageService}
}

func (h *PageHandler) Create(c *gin.Context) {
	var req models.CreatePageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(err, "Failed to parse create page request", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	page, err := h.pageService.Create(req)
	if err != nil {
		logger.Error(err, "Failed to create page", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create page"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"page": page})
}

func (h *PageHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page id"})
		return
	}

	var req models.UpdatePageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(err, "Failed to parse update page request", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	page, err := h.pageService.Update(uint(id), req)
	if err != nil {
		logger.Error(err, "Failed to update page", map[string]interface{}{"page_id": id})

		// Check for specific error types to return better messages
		errMsg := err.Error()
		if strings.Contains(errMsg, "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": errMsg})
			return
		}
		if strings.Contains(errMsg, "required") || strings.Contains(errMsg, "invalid") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"page": page})
}

func (h *PageHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page id"})
		return
	}

	if err := h.pageService.Delete(uint(id)); err != nil {
		logger.Error(err, "Failed to delete page", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete page"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "page deleted successfully"})
}

func (h *PageHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page id"})
		return
	}

	page, err := h.pageService.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"page": page})
}

func (h *PageHandler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	page, err := h.pageService.GetBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"page": page})
}

func (h *PageHandler) GetAll(c *gin.Context) {
	pages, err := h.pageService.GetAll()
	if err != nil {
		logger.Error(err, "Failed to retrieve all pages", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve pages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pages": pages})
}

func (h *PageHandler) GetAllAdmin(c *gin.Context) {
	pages, err := h.pageService.GetAllAdmin()
	if err != nil {
		logger.Error(err, "Failed to retrieve all admin pages", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve pages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pages": pages})
}

func (h *PageHandler) UpdateAllSectionPadding(c *gin.Context) {
	var req models.UpdateAllPageSectionsPaddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(err, "Failed to parse update section padding request", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	pagesUpdated, sectionsUpdated, padding, err := h.pageService.UpdateAllSectionPadding(req.PaddingVertical)
	if err != nil {
		logger.Error(err, "Failed to update all section padding", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update section padding"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pages_updated":    pagesUpdated,
		"sections_updated": sectionsUpdated,
		"padding_vertical": padding,
	})
}

func (h *PageHandler) PublishPage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page id"})
		return
	}

	if err := h.pageService.PublishPage(uint(id)); err != nil {
		logger.Error(err, "Failed to publish page", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish page"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "page published successfully"})
}

func (h *PageHandler) UnpublishPage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page id"})
		return
	}

	if err := h.pageService.UnpublishPage(uint(id)); err != nil {
		logger.Error(err, "Failed to unpublish page", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unpublish page"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "page unpublished successfully"})
}
