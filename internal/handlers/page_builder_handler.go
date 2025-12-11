package handlers

import (
	"net/http"
	"strconv"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

type PageBuilderHandler struct {
	pageService *service.PageService
}

func NewPageBuilderHandler(pageService *service.PageService) *PageBuilderHandler {
	return &PageBuilderHandler{
		pageService: pageService,
	}
}

// GetPageBuilder returns page data optimized for the page builder UI.
// GET /api/admin/pages/:id/builder
func (h *PageBuilderHandler) GetPageBuilder(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{
		"page": page,
		"meta": gin.H{
			"total_sections": len(page.Sections),
			"is_published":   page.Published,
		},
	})
}

// DuplicatePage creates a copy of an existing page.
// POST /api/admin/pages/:id/duplicate
func (h *PageBuilderHandler) DuplicatePage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page id"})
		return
	}

	page, err := h.pageService.DuplicatePage(uint(id))
	if err != nil {
		logger.Error(err, "Failed to duplicate page", map[string]interface{}{"page_id": id})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to duplicate page"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"page": page})
}

// ReorderSections updates the order of sections within a page.
// POST /api/admin/pages/:id/sections/reorder
func (h *PageBuilderHandler) ReorderSections(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page id"})
		return
	}

	var req struct {
		SectionIDs []string `json:"section_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	page, err := h.pageService.ReorderSections(uint(id), req.SectionIDs)
	if err != nil {
		logger.Error(err, "Failed to reorder sections", map[string]interface{}{"page_id": id})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder sections"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"page": page})
}

// AddSection adds a new section to a page.
// POST /api/admin/pages/:id/sections
func (h *PageBuilderHandler) AddSection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page id"})
		return
	}

	var req models.AddSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	page, err := h.pageService.AddSection(uint(id), req)
	if err != nil {
		logger.Error(err, "Failed to add section", map[string]interface{}{"page_id": id})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add section"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"page": page})
}

// UpdateSection updates an existing section within a page.
// PUT /api/admin/pages/:id/sections/:sectionId
func (h *PageBuilderHandler) UpdateSection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page id"})
		return
	}

	sectionID := c.Param("sectionId")
	if sectionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "section id is required"})
		return
	}

	var req models.UpdateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	page, err := h.pageService.UpdateSection(uint(id), sectionID, req)
	if err != nil {
		logger.Error(err, "Failed to update section", map[string]interface{}{
			"page_id":    id,
			"section_id": sectionID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update section"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"page": page})
}

// DeleteSection removes a section from a page.
// DELETE /api/admin/pages/:id/sections/:sectionId
func (h *PageBuilderHandler) DeleteSection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page id"})
		return
	}

	sectionID := c.Param("sectionId")
	if sectionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "section id is required"})
		return
	}

	page, err := h.pageService.DeleteSection(uint(id), sectionID)
	if err != nil {
		logger.Error(err, "Failed to delete section", map[string]interface{}{
			"page_id":    id,
			"section_id": sectionID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete section"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"page": page})
}

// DuplicateSection creates a copy of an existing section within a page.
// POST /api/admin/pages/:id/sections/:sectionId/duplicate
func (h *PageBuilderHandler) DuplicateSection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page id"})
		return
	}

	sectionID := c.Param("sectionId")
	if sectionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "section id is required"})
		return
	}

	page, err := h.pageService.DuplicateSection(uint(id), sectionID)
	if err != nil {
		logger.Error(err, "Failed to duplicate section", map[string]interface{}{
			"page_id":    id,
			"section_id": sectionID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to duplicate section"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"page": page})
}

// GetPageTemplates returns available page templates.
// GET /api/admin/pages/templates
func (h *PageBuilderHandler) GetPageTemplates(c *gin.Context) {
	templates := h.pageService.GetPageTemplates()
	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// CreateFromTemplate creates a new page from a template.
// POST /api/admin/pages/templates/:templateId
func (h *PageBuilderHandler) CreateFromTemplate(c *gin.Context) {
	templateID := c.Param("templateId")
	if templateID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "template id is required"})
		return
	}

	var req struct {
		Title string `json:"title" binding:"required"`
		Slug  string `json:"slug" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	page, err := h.pageService.CreateFromTemplate(templateID, req.Title, req.Slug)
	if err != nil {
		logger.Error(err, "Failed to create page from template", map[string]interface{}{
			"template_id": templateID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create page from template"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"page": page})
}

// PreviewPage returns page data formatted for preview mode.
// GET /api/admin/pages/:id/preview
func (h *PageBuilderHandler) PreviewPage(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{
		"page":         page,
		"preview_mode": true,
	})
}

// ValidatePageSlug checks if a slug is available.
// POST /api/admin/pages/validate-slug
func (h *PageBuilderHandler) ValidatePageSlug(c *gin.Context) {
	var req struct {
		Slug      string `json:"slug" binding:"required"`
		ExcludeID *uint  `json:"exclude_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	available, err := h.pageService.IsSlugAvailable(req.Slug, req.ExcludeID)
	if err != nil {
		logger.Error(err, "Failed to validate slug", map[string]interface{}{"slug": req.Slug})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate slug"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"available": available,
		"slug":      req.Slug,
	})
}

// GetPageBuilderConfig returns configuration for the page builder UI.
// GET /api/admin/pages/builder/config
func (h *PageBuilderHandler) GetPageBuilderConfig(c *gin.Context) {
	config := h.pageService.GetPageBuilderConfig()
	c.JSON(http.StatusOK, gin.H{"config": config})
}
