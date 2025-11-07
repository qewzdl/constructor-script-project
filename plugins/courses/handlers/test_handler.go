package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	courseservice "constructor-script-backend/plugins/courses/service"
)

type TestHandler struct {
	service *courseservice.TestService
}

func NewTestHandler(service *courseservice.TestService) *TestHandler {
	return &TestHandler{service: service}
}

func (h *TestHandler) SetService(service *courseservice.TestService) {
	if h == nil {
		return
	}
	h.service = service
}

func (h *TestHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "course test service unavailable"})
		return false
	}
	return true
}

func (h *TestHandler) Create(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	var req models.CreateCourseTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	test, err := h.service.Create(req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"test": test})
}

func (h *TestHandler) Update(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req models.UpdateCourseTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	test, err := h.service.Update(id, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"test": test})
}

func (h *TestHandler) Delete(c *gin.Context) {
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

func (h *TestHandler) Get(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	test, err := h.service.GetByID(id)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"test": test})
}

func (h *TestHandler) List(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	tests, err := h.service.List()
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"tests": tests})
}

func (h *TestHandler) Submit(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var req models.SubmitCourseTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.Submit(id, userID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": result})
}

func (h *TestHandler) writeError(c *gin.Context, err error) {
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
