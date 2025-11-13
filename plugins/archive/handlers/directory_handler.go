package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"constructor-script-backend/internal/models"
	archiveservice "constructor-script-backend/plugins/archive/service"
)

type DirectoryHandler struct {
	service *archiveservice.DirectoryService
}

func NewDirectoryHandler(service *archiveservice.DirectoryService) *DirectoryHandler {
	return &DirectoryHandler{service: service}
}

func (h *DirectoryHandler) SetService(service *archiveservice.DirectoryService) {
	if h == nil {
		return
	}
	h.service = service
}

func (h *DirectoryHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.service == nil {
		if c != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "archive plugin is not active"})
		}
		return false
	}
	return true
}

func (h *DirectoryHandler) List(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	view := strings.TrimSpace(strings.ToLower(c.Query("view")))
	treeFlag := c.Query("tree")
	if view == "tree" || strings.EqualFold(treeFlag, "true") || treeFlag == "1" {
		directories, err := h.service.ListTree(true)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"directories": directories})
		return
	}

	var parentID *uint
	if raw := strings.TrimSpace(c.Query("parent_id")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_id"})
			return
		}
		converted := uint(value)
		parentID = &converted
	}

	directories, err := h.service.ListByParent(parentID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"directories": directories})
}

func (h *DirectoryHandler) Get(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid directory id"})
		return
	}

	directory, err := h.service.GetByID(uint(id), true)
	if err != nil {
		if errors.Is(err, archiveservice.ErrDirectoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "directory not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"directory": directory})
}

func (h *DirectoryHandler) Create(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	var req models.CreateArchiveDirectoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	directory, err := h.service.Create(req)
	if err != nil {
		switch {
		case errors.Is(err, archiveservice.ErrInvalidParent):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, archiveservice.ErrSlugConflict):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"directory": directory})
}

func (h *DirectoryHandler) Update(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid directory id"})
		return
	}

	var req models.UpdateArchiveDirectoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	directory, err := h.service.Update(uint(id), req)
	if err != nil {
		switch {
		case errors.Is(err, archiveservice.ErrDirectoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "directory not found"})
		case errors.Is(err, archiveservice.ErrInvalidParent):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, archiveservice.ErrSlugConflict):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"directory": directory})
}

func (h *DirectoryHandler) Delete(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid directory id"})
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		switch {
		case errors.Is(err, archiveservice.ErrDirectoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "directory not found"})
		case errors.Is(err, archiveservice.ErrDirectoryNotEmpty):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
