package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	courseservice "constructor-script-backend/plugins/courses/service"
)

type VideoHandler struct {
	service *courseservice.VideoService
}

func NewVideoHandler(service *courseservice.VideoService) *VideoHandler {
	return &VideoHandler{service: service}
}

func (h *VideoHandler) SetService(service *courseservice.VideoService) {
	if h == nil {
		return
	}
	h.service = service
}

func (h *VideoHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "course video service unavailable"})
		return false
	}
	return true
}

func (h *VideoHandler) Create(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	var req models.CreateCourseVideoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if rawSections := c.PostForm("sections"); rawSections != "" {
		if err := json.Unmarshal([]byte(rawSections), &req.Sections); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sections payload"})
			return
		}
	}

	if rawAttachments := c.PostForm("attachments"); rawAttachments != "" {
		if err := json.Unmarshal([]byte(rawAttachments), &req.Attachments); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid attachments payload"})
			return
		}
	}

	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video file is required"})
		return
	}

	video, err := h.service.Create(c.Request.Context(), req, file)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"video": video})
}

func (h *VideoHandler) Update(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req models.UpdateCourseVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	video, err := h.service.Update(id, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"video": video})
}

func (h *VideoHandler) UpdateSubtitle(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req models.UpdateCourseVideoSubtitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	video, err := h.service.UpdateSubtitle(c.Request.Context(), id, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"video": video})
}

func (h *VideoHandler) Delete(c *gin.Context) {
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

func (h *VideoHandler) Get(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	video, err := h.service.GetByID(id)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"video": video})
}

func (h *VideoHandler) List(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	videos, err := h.service.List()
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"videos": videos})
}

func (h *VideoHandler) writeError(c *gin.Context, err error) {
	switch {
	case courseservice.IsValidationError(err):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
