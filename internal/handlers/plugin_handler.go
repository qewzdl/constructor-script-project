package handlers

import (
	"errors"
	"net/http"

	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

type PluginHandler struct {
	service *service.PluginService
}

func NewPluginHandler(service *service.PluginService) *PluginHandler {
	return &PluginHandler{service: service}
}

func (h *PluginHandler) List(c *gin.Context) {
	ctx := c.Request.Context()

	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "plugin service unavailable"})
		return
	}

	plugins, err := h.service.List()
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrPluginManagerUnavailable) {
			status = http.StatusServiceUnavailable
		}
		logger.ErrorContext(ctx, err, "Failed to list plugins", nil)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"plugins": plugins})
}

func (h *PluginHandler) Install(c *gin.Context) {
	ctx := c.Request.Context()

	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "plugin service unavailable"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugin archive is required"})
		return
	}
	defer file.Close()

	info, err := h.service.Install(file, header.Size, header.Filename)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrPluginManagerUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, service.ErrPluginRepositoryUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, service.ErrInvalidPluginPackage):
			status = http.StatusBadRequest
		}
		logger.ErrorContext(ctx, err, "Failed to install plugin", map[string]interface{}{"filename": header.Filename})
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"plugin": info})
}

func (h *PluginHandler) Activate(c *gin.Context) {
	ctx := c.Request.Context()

	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "plugin service unavailable"})
		return
	}

	slug := c.Param("slug")
	info, err := h.service.Activate(slug)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrPluginManagerUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, service.ErrPluginRepositoryUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, service.ErrPluginNotFound):
			status = http.StatusNotFound
		}
		logger.ErrorContext(ctx, err, "Failed to activate plugin", map[string]interface{}{"slug": slug})
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"plugin": info})
}

func (h *PluginHandler) Deactivate(c *gin.Context) {
	ctx := c.Request.Context()

	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "plugin service unavailable"})
		return
	}

	slug := c.Param("slug")
	info, err := h.service.Deactivate(slug)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrPluginManagerUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, service.ErrPluginRepositoryUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, service.ErrPluginNotFound):
			status = http.StatusNotFound
		}
		logger.ErrorContext(ctx, err, "Failed to deactivate plugin", map[string]interface{}{"slug": slug})
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"plugin": info})
}

func (h *PluginHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "plugin service unavailable"})
		return
	}

	slug := c.Param("slug")
	info, err := h.service.Delete(slug)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrPluginManagerUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, service.ErrPluginRepositoryUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, service.ErrPluginNotFound):
			status = http.StatusNotFound
		}
		logger.ErrorContext(ctx, err, "Failed to delete plugin", map[string]interface{}{"slug": slug})
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"plugin": info})
}
