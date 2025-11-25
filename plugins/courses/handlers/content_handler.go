package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	courseservice "constructor-script-backend/plugins/courses/service"
)

type ContentHandler struct {
	service *courseservice.ContentService
}

func NewContentHandler(service *courseservice.ContentService) *ContentHandler {
	return &ContentHandler{service: service}
}

func (h *ContentHandler) SetService(service *courseservice.ContentService) {
	if h == nil {
		return
	}
	h.service = service
}

func (h *ContentHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "course content service unavailable"})
		return false
	}
	return true
}

func (h *ContentHandler) Create(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	var req models.CreateCourseContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	content, err := h.service.Create(req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"content": content})
}

func (h *ContentHandler) Update(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req models.UpdateCourseContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	content, err := h.service.Update(id, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": content})
}

func (h *ContentHandler) Delete(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	if err := h.service.Delete(id); err != nil {
		h.writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ContentHandler) Get(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	content, err := h.service.GetByID(id)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": content})
}

func (h *ContentHandler) List(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	contents, err := h.service.List()
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"contents": contents})
}

func (h *ContentHandler) writeError(c *gin.Context, err error) {
	switch {
	case courseservice.IsValidationError(err):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
