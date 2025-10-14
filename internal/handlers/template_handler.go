package handlers

import (
	"fmt"
	"html/template"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/navigation"

	"github.com/microcosm-cc/bluemonday"
)

type TemplateHandler struct {
	postService      *service.PostService
	pageService      *service.PageService
	authService      *service.AuthService
	commentService   *service.CommentService
	searchService    *service.SearchService
	setupService     *service.SetupService
	templates        *template.Template
	config           *config.Config
	sanitizer        *bluemonday.Policy
	sectionRenderers map[string]SectionRenderer
	navigation       []navigation.Item
	blogEnabled      bool
}

func NewTemplateHandler(postService *service.PostService, pageService *service.PageService, authService *service.AuthService, commentService *service.CommentService, searchService *service.SearchService, setupService *service.SetupService, cfg *config.Config, templates *template.Template) (*TemplateHandler, error) {
	if templates == nil {
		return nil, fmt.Errorf("templates are required")
	}

	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class", "id").Globally()
	policy.AllowAttrs("style").OnElements("span", "div", "p")

	handler := &TemplateHandler{
		postService:    postService,
		pageService:    pageService,
		authService:    authService,
		commentService: commentService,
		searchService:  searchService,
		setupService:   setupService,
		templates:      templates,
		config:         cfg,
		sanitizer:      policy,
	}

	handler.registerDefaultSectionRenderers()

	return handler, nil
}

func (h *TemplateHandler) SetNavigation(items []navigation.Item) {
	h.navigation = items
}

func (h *TemplateHandler) EnableBlogModule() {
	h.blogEnabled = true
}
