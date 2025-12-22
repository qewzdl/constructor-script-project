package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	courseservice "constructor-script-backend/plugins/courses/service"
)

type PackageHandler struct {
	service    *courseservice.PackageService
	protection *courseservice.MaterialProtection
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

// SetMaterialProtection configures the signer used to protect course assets in responses.
func (h *PackageHandler) SetMaterialProtection(protection *courseservice.MaterialProtection) {
	if h == nil {
		return
	}
	h.protection = protection
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

func (h *PackageHandler) GrantToUser(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	packageID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req models.GrantCoursePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID := c.GetUint("user_id")

	access, err := h.service.GrantToUser(packageID, req, adminID)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"access": access})
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

	identifier := strings.TrimSpace(c.Param("id"))
	if identifier == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "course package not found"})
		return
	}

	pkg, err := h.service.GetByIdentifier(identifier)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"package": pkg})
}

func (h *PackageHandler) GetForUser(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	identifier := strings.TrimSpace(c.Param("id"))
	if identifier == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "course package not found"})
		return
	}

	if h.protection == nil || !h.protection.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "course materials are temporarily unavailable"})
		return
	}

	course, err := h.service.GetForUserByIdentifier(identifier, userID)
	if err != nil {
		h.writeError(c, err)
		return
	}

	if h.protection != nil {
		course = h.protection.ProtectCourseForUser(course, userID)
	}

	c.JSON(http.StatusOK, gin.H{"course": course})
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
