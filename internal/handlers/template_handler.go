package handlers

import (
	"errors"
	"html/template"
	"path/filepath"
	"sync"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/internal/theme"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/utils"

	"github.com/microcosm-cc/bluemonday"
)

type TemplateHandler struct {
	postService       *service.PostService
	categoryService   *service.CategoryService
	pageService       *service.PageService
	authService       *service.AuthService
	commentService    *service.CommentService
	searchService     *service.SearchService
	setupService      *service.SetupService
	socialLinkService *service.SocialLinkService
	menuService       *service.MenuService
	templates         *template.Template
	templatesMu       sync.RWMutex
	currentTheme      string
	themeManager      *theme.Manager
	config            *config.Config
	sanitizer         *bluemonday.Policy
	sectionRenderers  map[string]SectionRenderer
}

func NewTemplateHandler(
	postService *service.PostService,
	pageService *service.PageService,
	authService *service.AuthService,
	commentService *service.CommentService,
	searchService *service.SearchService,
	setupService *service.SetupService,
	categoryService *service.CategoryService,
	socialLinkService *service.SocialLinkService,
	menuService *service.MenuService,
	cfg *config.Config,
	themeManager *theme.Manager,
) (*TemplateHandler, error) {
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class", "id").Globally()
	policy.AllowAttrs("style").OnElements("span", "div", "p")

	handler := &TemplateHandler{
		postService:       postService,
		categoryService:   categoryService,
		pageService:       pageService,
		authService:       authService,
		commentService:    commentService,
		searchService:     searchService,
		setupService:      setupService,
		socialLinkService: socialLinkService,
		menuService:       menuService,
		themeManager:      themeManager,
		config:            cfg,
		sanitizer:         policy,
	}

	handler.registerDefaultSectionRenderers()

	if err := handler.reloadTemplates(); err != nil {
		return nil, err
	}

	return handler, nil
}

func (h *TemplateHandler) reloadTemplates() error {
	if h.themeManager == nil {
		return errors.New("theme manager not configured")
	}

	active := h.themeManager.Active()
	if active == nil {
		return errors.New("no active theme")
	}

	tmpl := template.New("").Funcs(utils.GetTemplateFuncs(h.themeManager.AssetModTime))
	templates, err := tmpl.ParseGlob(filepath.Join(active.TemplatesDir, "*.html"))
	if err != nil {
		return err
	}

	h.templatesMu.Lock()
	h.templates = templates
	h.currentTheme = active.Slug
	h.templatesMu.Unlock()

	logger.Info("Loaded templates", map[string]interface{}{"theme": active.Slug})
	return nil
}

func (h *TemplateHandler) templateClone() (*template.Template, error) {
	if h.themeManager == nil {
		return nil, errors.New("theme manager not configured")
	}

	active := h.themeManager.Active()
	if active == nil {
		return nil, errors.New("no active theme")
	}

	needsReload := false

	h.templatesMu.RLock()
	if h.templates == nil || h.currentTheme != active.Slug {
		needsReload = true
	}
	h.templatesMu.RUnlock()

	if needsReload {
		if err := h.reloadTemplates(); err != nil {
			return nil, err
		}
	}

	h.templatesMu.RLock()
	defer h.templatesMu.RUnlock()
	if h.templates == nil {
		return nil, errors.New("templates not loaded")
	}

	return h.templates.Clone()
}
