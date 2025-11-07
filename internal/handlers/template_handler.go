package handlers

import (
	"errors"
	"html/template"
	"net/http"
	"path/filepath"
	"sync"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/sections"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/internal/theme"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/utils"
	blogservice "constructor-script-backend/plugins/blog/service"
	courseservice "constructor-script-backend/plugins/courses/service"
	languageservice "constructor-script-backend/plugins/language/service"

	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
)

type TemplateHandler struct {
	postService        *blogservice.PostService
	categoryService    *blogservice.CategoryService
	pageService        *service.PageService
	authService        *service.AuthService
	commentService     *blogservice.CommentService
	searchService      *blogservice.SearchService
	setupService       *service.SetupService
	homepageService    *service.HomepageService
	languageService    *languageservice.LanguageService
	socialLinkService  *service.SocialLinkService
	menuService        *service.MenuService
	advertisingService *service.AdvertisingService
	coursePackageSvc   *courseservice.PackageService
	courseCheckoutSvc  *courseservice.CheckoutService
	fontService        *service.FontService
	templates          *template.Template
	templatesMu        sync.RWMutex
	currentTheme       string
	themeManager       *theme.Manager
	config             *config.Config
	sanitizer          *bluemonday.Policy
	sectionRegistry    *sections.Registry
}

func NewTemplateHandler(
	postService *blogservice.PostService,
	pageService *service.PageService,
	authService *service.AuthService,
	commentService *blogservice.CommentService,
	searchService *blogservice.SearchService,
	setupService *service.SetupService,
	languageService *languageservice.LanguageService,
	homepageService *service.HomepageService,
	categoryService *blogservice.CategoryService,
	socialLinkService *service.SocialLinkService,
	menuService *service.MenuService,
	fontService *service.FontService,
	advertisingService *service.AdvertisingService,
	coursePackageService *courseservice.PackageService,
	courseCheckoutService *courseservice.CheckoutService,
	cfg *config.Config,
	themeManager *theme.Manager,
) (*TemplateHandler, error) {
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class", "id").Globally()
	policy.AllowAttrs("style").OnElements("span", "div", "p")

	handler := &TemplateHandler{
		postService:        postService,
		categoryService:    categoryService,
		pageService:        pageService,
		authService:        authService,
		commentService:     commentService,
		searchService:      searchService,
		setupService:       setupService,
		languageService:    languageService,
		homepageService:    homepageService,
		socialLinkService:  socialLinkService,
		menuService:        menuService,
		fontService:        fontService,
		advertisingService: advertisingService,
		coursePackageSvc:   coursePackageService,
		courseCheckoutSvc:  courseCheckoutService,
		themeManager:       themeManager,
		config:             cfg,
		sanitizer:          policy,
	}

	handler.sectionRegistry = sections.DefaultRegistry()

	if err := handler.reloadTemplates(); err != nil {
		return nil, err
	}

	return handler, nil
}

// SetBlogServices swaps the blog-related services used by the template handler.
func (h *TemplateHandler) SetBlogServices(
	postService *blogservice.PostService,
	categoryService *blogservice.CategoryService,
	commentService *blogservice.CommentService,
	searchService *blogservice.SearchService,
) {
	if h == nil {
		return
	}

	h.postService = postService
	h.categoryService = categoryService
	h.commentService = commentService
	h.searchService = searchService
}

// SetLanguageService updates the language service dependency used by the template handler.
func (h *TemplateHandler) SetLanguageService(languageService *languageservice.LanguageService) {
	if h == nil {
		return
	}
	h.languageService = languageService
}

// SetCoursePackageService updates the course package service dependency used by the template handler.
func (h *TemplateHandler) SetCoursePackageService(packageService *courseservice.PackageService) {
	if h == nil {
		return
	}
	h.coursePackageSvc = packageService
}

// SetCourseCheckoutService updates the course checkout service dependency used by the template handler.
func (h *TemplateHandler) SetCourseCheckoutService(checkoutService *courseservice.CheckoutService) {
	if h == nil {
		return
	}
	h.courseCheckoutSvc = checkoutService
}

func (h *TemplateHandler) blogEnabled() bool {
	return h != nil && h.postService != nil
}

func (h *TemplateHandler) coursesEnabled() bool {
	return h != nil && h.coursePackageSvc != nil
}

func (h *TemplateHandler) courseCheckoutEnabled() bool {
	return h != nil && h.courseCheckoutSvc != nil && h.courseCheckoutSvc.Enabled()
}

func (h *TemplateHandler) ensureBlogAvailable(c *gin.Context) bool {
	if h == nil || h.postService == nil {
		if c != nil {
			h.renderError(c, http.StatusServiceUnavailable, "Blog unavailable", "The blog plugin is not active.")
		}
		return false
	}
	return true
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

func (h *TemplateHandler) ReloadTemplates() error {
	if h == nil {
		return errors.New("template handler not configured")
	}
	return h.reloadTemplates()
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

// SanitizeHTML makes TemplateHandler compatible with sections.RenderContext.
func (h *TemplateHandler) SanitizeHTML(input string) string {
	if h == nil || h.sanitizer == nil {
		return input
	}
	return h.sanitizer.Sanitize(input)
}

// CloneTemplates makes TemplateHandler compatible with sections.RenderContext.
func (h *TemplateHandler) CloneTemplates() (*template.Template, error) {
	return h.templateClone()
}
