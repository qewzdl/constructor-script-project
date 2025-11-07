package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	courseservice "constructor-script-backend/plugins/courses/service"
)

type TopicHandler struct {
	service *courseservice.TopicService
}

func NewTopicHandler(service *courseservice.TopicService) *TopicHandler {
	return &TopicHandler{service: service}
}

func (h *TopicHandler) SetService(service *courseservice.TopicService) {
	if h == nil {
		return
	}
	h.service = service
}

func (h *TopicHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "course topic service unavailable"})
		return false
	}
	return true
}

func (h *TopicHandler) Create(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	var req models.CreateCourseTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topic, err := h.service.Create(req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"topic": topic})
}

func (h *TopicHandler) Update(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req models.UpdateCourseTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topic, err := h.service.Update(id, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"topic": topic})
}

func (h *TopicHandler) UpdateVideos(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req models.ReorderCourseTopicVideosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topic, err := h.service.UpdateVideos(id, req.VideoIDs)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"topic": topic})
}

func (h *TopicHandler) UpdateSteps(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req models.UpdateCourseTopicStepsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topic, err := h.service.UpdateSteps(id, req.Steps)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"topic": topic})
}

func (h *TopicHandler) Delete(c *gin.Context) {
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

func (h *TopicHandler) Get(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	topic, err := h.service.GetByID(id)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"topic": topic})
}

func (h *TopicHandler) List(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	topics, err := h.service.List()
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"topics": topics})
}

func (h *TopicHandler) writeError(c *gin.Context, err error) {
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
