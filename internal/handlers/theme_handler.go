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
	theme, needsInitialization, err := h.service.Activate(slug)
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

	if needsInitialization && (h.pageService != nil || h.menuService != nil) {
		if activeTheme, activeErr := h.service.ActiveTheme(); activeErr == nil {
			if h.pageService != nil {
				seed.EnsureDefaultPages(h.pageService, activeTheme.PagesFS())
			}
			if h.menuService != nil {
				seed.EnsureDefaultMenu(h.menuService, activeTheme.MenuFS())
			}
			if err := h.service.MarkInitialized(activeTheme.Slug); err != nil {
				logger.Error(err, "Failed to mark theme defaults as applied", map[string]interface{}{"theme": activeTheme.Slug})
			}
		} else {
			logger.Error(activeErr, "Failed to load active theme for defaults", nil)
		}
	} else if needsInitialization {
		if err := h.service.MarkInitialized(theme.Slug); err != nil {
			logger.Error(err, "Failed to mark theme defaults as applied", map[string]interface{}{"theme": theme.Slug})
		}
	}

	c.JSON(http.StatusOK, gin.H{"theme": theme})
}

func (h *ThemeHandler) Reload(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "theme service unavailable"})
		return
	}

	slug := c.Param("slug")
	theme, _, err := h.service.Activate(slug)
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
		logger.Error(err, "Failed to reload theme", map[string]interface{}{"slug": slug})
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	activeTheme, activeErr := h.service.ActiveTheme()
	if activeErr != nil {
		logger.Error(activeErr, "Failed to resolve active theme during reload", map[string]interface{}{"slug": slug})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve active theme"})
		return
	}

	var errs []error

	if h.pageService != nil {
		if err := seed.ResetPages(h.pageService, activeTheme.PagesFS()); err != nil {
			logger.Error(err, "Failed to reset theme pages", map[string]interface{}{"theme": activeTheme.Slug})
			errs = append(errs, err)
		}
	}

	if h.menuService != nil {
		if err := seed.ResetMenu(h.menuService, activeTheme.MenuFS()); err != nil {
			logger.Error(err, "Failed to reset theme menu", map[string]interface{}{"theme": activeTheme.Slug})
			errs = append(errs, err)
		}
	}

	if combined := errors.Join(errs...); combined != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reload theme defaults"})
		return
	}

	if err := h.service.MarkInitialized(activeTheme.Slug); err != nil {
		logger.Error(err, "Failed to mark theme defaults as applied", map[string]interface{}{"theme": activeTheme.Slug})
	}

	c.JSON(http.StatusOK, gin.H{"theme": theme})
}
