package handlers

import (
	"net/http"

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

	metadata := h.SectionMetadata()
	if metadata == nil {
		// Return basic info if metadata not available
		c.JSON(http.StatusOK, gin.H{
			"sections": []string{
				"paragraph",
				"image",
				"image_group",
				"file_group",
				"list",
				"search",
				"posts_list",
				"categories_list",
				"courses_list",
				"grid",
				"standard",
			},
			"has_metadata": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sections":     metadata,
		"has_metadata": true,
	})
}
