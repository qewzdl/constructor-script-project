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

type FileHandler struct {
	service *archiveservice.FileService
}

func NewFileHandler(service *archiveservice.FileService) *FileHandler {
	return &FileHandler{service: service}
}

func (h *FileHandler) SetService(service *archiveservice.FileService) {
	if h == nil {
		return
	}
	h.service = service
}

func (h *FileHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.service == nil {
		if c != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "archive plugin is not active"})
		}
		return false
	}
	return true
}

func (h *FileHandler) List(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	raw := strings.TrimSpace(c.Query("directory_id"))
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "directory_id is required"})
		return
	}
	directoryID, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid directory_id"})
		return
	}

	files, err := h.service.ListByDirectory(uint(directoryID), true)
	if err != nil {
		if errors.Is(err, archiveservice.ErrDirectoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "directory not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

func (h *FileHandler) Get(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	file, err := h.service.GetByID(uint(id), true)
	if err != nil {
		if errors.Is(err, archiveservice.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"file": file})
}

func (h *FileHandler) Create(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	var req models.CreateArchiveFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file, err := h.service.Create(req)
	if err != nil {
		switch {
		case errors.Is(err, archiveservice.ErrDirectoryNotFound):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"file": file})
}

func (h *FileHandler) Update(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	var req models.UpdateArchiveFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file, err := h.service.Update(uint(id), req)
	if err != nil {
		switch {
		case errors.Is(err, archiveservice.ErrFileNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		case errors.Is(err, archiveservice.ErrDirectoryNotFound):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"file": file})
}

func (h *FileHandler) Delete(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if errors.Is(err, archiveservice.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
