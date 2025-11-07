package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	courseservice "constructor-script-backend/plugins/courses/service"
)

type PackageHandler struct {
	service *courseservice.PackageService
}

func NewPackageHandler(service *courseservice.PackageService) *PackageHandler {
	return &PackageHandler{service: service}
}

func (h *PackageHandler) SetService(service *courseservice.PackageService) {
	if h == nil {
		return
	}
	h.service = service
}

func (h *PackageHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "course package service unavailable"})
		return false
	}
	return true
}

func (h *PackageHandler) Create(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	var req models.CreateCoursePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pkg, err := h.service.Create(req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"package": pkg})
}

func (h *PackageHandler) Update(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req models.UpdateCoursePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pkg, err := h.service.Update(id, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"package": pkg})
}

func (h *PackageHandler) UpdateTopics(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req models.ReorderCoursePackageTopicsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pkg, err := h.service.UpdateTopics(id, req.TopicIDs)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"package": pkg})
}

func (h *PackageHandler) Delete(c *gin.Context) {
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

func (h *PackageHandler) Get(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	pkg, err := h.service.GetByID(id)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"package": pkg})
}

func (h *PackageHandler) List(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	pkgs, err := h.service.List()
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"packages": pkgs})
}

func (h *PackageHandler) writeError(c *gin.Context, err error) {
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
