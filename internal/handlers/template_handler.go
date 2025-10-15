package handlers

import (
	"html/template"
	"path/filepath"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/utils"

	"github.com/microcosm-cc/bluemonday"
)

type TemplateHandler struct {
	postService      *service.PostService
	categoryService  *service.CategoryService
	pageService      *service.PageService
	authService      *service.AuthService
	commentService   *service.CommentService
	searchService    *service.SearchService
	setupService     *service.SetupService
	templates        *template.Template
	config           *config.Config
	sanitizer        *bluemonday.Policy
	sectionRenderers map[string]SectionRenderer
}

func NewTemplateHandler(
	postService *service.PostService,
	pageService *service.PageService,
	authService *service.AuthService,
	commentService *service.CommentService,
	searchService *service.SearchService,
	setupService *service.SetupService,
	categoryService *service.CategoryService,
	cfg *config.Config,
	templatesDir string,
) (*TemplateHandler, error) {
	tmpl := template.New("").Funcs(utils.GetTemplateFuncs())
	templates, err := tmpl.ParseGlob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		return nil, err
	}

	logger.Info("Loaded templates:", map[string]interface{}{
		"templates": templates.DefinedTemplates(),
	})

	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class", "id").Globally()
	policy.AllowAttrs("style").OnElements("span", "div", "p")

	handler := &TemplateHandler{
		postService:     postService,
		categoryService: categoryService,
		pageService:     pageService,
		authService:     authService,
		commentService:  commentService,
		searchService:   searchService,
		setupService:    setupService,
		templates:       templates,
		config:          cfg,
		sanitizer:       policy,
	}

	handler.registerDefaultSectionRenderers()

	return handler, nil
}
