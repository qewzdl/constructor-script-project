package handlers

import (
	"errors"
	"net/http"

	"constructor-script-backend/internal/seed"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

type ThemeHandler struct {
	service     *service.ThemeService
	pageService *service.PageService
	menuService *service.MenuService
}

func NewThemeHandler(themeService *service.ThemeService, pageService *service.PageService, menuService *service.MenuService) *ThemeHandler {
	return &ThemeHandler{service: themeService, pageService: pageService, menuService: menuService}
}

func (h *ThemeHandler) List(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "theme service unavailable"})
		return
	}

	themes, err := h.service.List()
	if err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrThemeManagerUnavailable {
			status = http.StatusServiceUnavailable
		}
		logger.Error(err, "Failed to list themes", nil)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"themes": themes})
}

func (h *ThemeHandler) Activate(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "theme service unavailable"})
		return
	}

	slug := c.Param("slug")
	theme, err := h.service.Activate(slug)
	if err != nil {
		status := http.StatusInternalServerError
		switch err {
		case service.ErrThemeManagerUnavailable:
			status = http.StatusServiceUnavailable
		default:
			if errors.Is(err, service.ErrThemeNotFound) {
				status = http.StatusNotFound
			}
		}
		logger.Error(err, "Failed to activate theme", map[string]interface{}{"slug": slug})
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	if h.pageService != nil || h.menuService != nil {
		if activeTheme, activeErr := h.service.ActiveTheme(); activeErr == nil {
			if h.pageService != nil {
				seed.EnsureDefaultPages(h.pageService, activeTheme.PagesFS())
			}
			if h.menuService != nil {
				seed.EnsureDefaultMenu(h.menuService, activeTheme.MenuFS())
			}
		} else {
			logger.Error(activeErr, "Failed to load active theme for defaults", nil)
		}
	}

	c.JSON(http.StatusOK, gin.H{"theme": theme})
}
