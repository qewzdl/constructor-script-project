package blog

import (
	"fmt"
	"html/template"
	"path/filepath"

	"constructor-script-backend/internal/handlers"
	"constructor-script-backend/pkg/navigation"

	"github.com/gin-gonic/gin"
)

type Options struct {
	Handler      *handlers.TemplateHandler
	TemplatesDir string
}

type Module struct {
	handler      *handlers.TemplateHandler
	templatesDir string
}

func New(opts Options) (*Module, error) {
	if opts.Handler == nil {
		return nil, fmt.Errorf("template handler is required")
	}

	templatesDir := opts.TemplatesDir
	if templatesDir == "" {
		templatesDir = filepath.Join("internal", "modules", "blog", "templates")
	}

	opts.Handler.EnableBlogModule()

	return &Module{
		handler:      opts.Handler,
		templatesDir: templatesDir,
	}, nil
}

func (m *Module) Name() string {
	return "blog"
}

func (m *Module) RegisterTemplates(tmpl *template.Template) error {
	patterns := []string{
		filepath.Join(m.templatesDir, "*.html"),
		filepath.Join(m.templatesDir, "components", "*.html"),
	}

	for _, pattern := range patterns {
		if _, err := tmpl.ParseGlob(pattern); err != nil {
			return err
		}
	}

	return nil
}

func (m *Module) Mount(router *gin.Engine) {
	router.GET("/blog", m.handler.RenderBlog)
	router.GET("/blog/post/:slug", m.handler.RenderPost)
	router.GET("/category/:slug", m.handler.RenderCategory)
	router.GET("/tag/:slug", m.handler.RenderTag)
}

func (m *Module) NavigationItems() []navigation.Item {
	return []navigation.Item{{
		Label: "Blog",
		Path:  "/blog",
	}}
}
