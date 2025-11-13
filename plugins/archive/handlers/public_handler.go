package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"constructor-script-backend/internal/models"
	archiveservice "constructor-script-backend/plugins/archive/service"
)

type PublicHandler struct {
	directoryService *archiveservice.DirectoryService
	fileService      *archiveservice.FileService
}

func NewPublicHandler(directoryService *archiveservice.DirectoryService, fileService *archiveservice.FileService) *PublicHandler {
	return &PublicHandler{directoryService: directoryService, fileService: fileService}
}

func (h *PublicHandler) SetServices(directoryService *archiveservice.DirectoryService, fileService *archiveservice.FileService) {
	if h == nil {
		return
	}
	h.directoryService = directoryService
	h.fileService = fileService
}

func (h *PublicHandler) ensureServices(c *gin.Context) bool {
	if h == nil || h.directoryService == nil || h.fileService == nil {
		if c != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "archive plugin is not active"})
		}
		return false
	}
	return true
}

func (h *PublicHandler) Tree(c *gin.Context) {
	if !h.ensureServices(c) {
		return
	}

	directories, err := h.directoryService.ListPublishedTree()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"directories": directories})
}

func (h *PublicHandler) GetDirectory(c *gin.Context) {
	if !h.ensureServices(c) {
		return
	}

	rawPath := strings.Trim(c.Param("path"), "/")
	if rawPath == "" {
		directories, err := h.directoryService.ListPublishedTree()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"directories": directories})
		return
	}

	directory, err := h.directoryService.GetByPath(rawPath, false)
	if err != nil {
		if errors.Is(err, archiveservice.ErrDirectoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "directory not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	files, err := h.fileService.ListByDirectory(directory.ID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	children, err := h.directoryService.ListByParent(&directory.ID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	breadcrumbs, err := h.directoryService.BuildBreadcrumbs(rawPath, false)
	if err != nil {
		if errors.Is(err, archiveservice.ErrDirectoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "directory not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"directory":   directory,
		"files":       files,
		"children":    children,
		"breadcrumbs": breadcrumbs,
	})
}

func (h *PublicHandler) GetFile(c *gin.Context) {
	if !h.ensureServices(c) {
		return
	}

	rawPath := strings.Trim(c.Param("path"), "/")
	if rawPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file path is required"})
		return
	}

	file, err := h.fileService.GetByPath(rawPath, false)
	if err != nil {
		if errors.Is(err, archiveservice.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	segments := strings.Split(rawPath, "/")
	if len(segments) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
		return
	}

	directoryPath := strings.Join(segments[:len(segments)-1], "/")
	directory, err := h.directoryService.GetByPath(directoryPath, false)
	if err != nil {
		if errors.Is(err, archiveservice.ErrDirectoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "directory not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	breadcrumbs, err := h.directoryService.BuildBreadcrumbs(directoryPath, false)
	if err != nil {
		if errors.Is(err, archiveservice.ErrDirectoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "directory not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	breadcrumbs = append(breadcrumbs, models.ArchiveBreadcrumb{Name: strings.TrimSpace(file.Name), Path: file.Path})

	c.JSON(http.StatusOK, gin.H{
		"file":         file,
		"directory":    directory,
		"breadcrumbs":  breadcrumbs,
		"download_url": file.FileURL,
	})
}
