package handlers

import (
	"net/http"

	"constructor-script-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// GetAvailableSections returns metadata for all registered section types.
// This endpoint is useful for admin interfaces to dynamically discover available sections.
// GET /api/admin/sections/available
func (h *TemplateHandler) GetAvailableSections(c *gin.Context) {
	if h == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Handler not initialized"})
		return
	}

	options := service.SectionCatalogOptions{
		BlogEnabled:    h.blogEnabled(),
		CoursesEnabled: h.coursesEnabled(),
	}

	var catalog *service.SectionCatalog
	if h.pageService != nil {
		catalog = h.pageService.SectionCatalog(options)
	} else {
		catalog = service.NewSectionCatalog(h.themeManager, options)
	}

	metadata := h.SectionMetadata()
	response := gin.H{
		"sections":            catalog.SectionTypeConfigs(),
		"section_definitions": catalog.SectionDefinitions(),
		"element_definitions": catalog.ElementDefinitions(),
		"has_metadata":        len(metadata) > 0,
	}
	if len(metadata) > 0 {
		response["metadata"] = metadata
	}

	c.JSON(http.StatusOK, response)
}
