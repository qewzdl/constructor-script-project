package modules

import (
	"fmt"
	"html/template"

	"constructor-script-backend/pkg/navigation"

	"github.com/gin-gonic/gin"
)

// Module describes a self-contained feature that can register its own routes,
// templates, and navigation entries. Modules allow optional functionality such
// as the blog to live in isolated packages that can be removed without touching
// the main application wiring.
type Module interface {
	Name() string
	RegisterTemplates(tmpl *template.Template) error
	Mount(router *gin.Engine)
	NavigationItems() []navigation.Item
}

// Manager keeps track of registered modules and applies their contributions to
// the application router and template engine.
type Manager struct {
	modules []Module
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Register(module Module) {
	if module == nil {
		return
	}

	m.modules = append(m.modules, module)
}

func (m *Manager) LoadTemplates(tmpl *template.Template) error {
	if tmpl == nil {
		return fmt.Errorf("template engine is required")
	}

	for _, module := range m.modules {
		if err := module.RegisterTemplates(tmpl); err != nil {
			return fmt.Errorf("module %s: %w", module.Name(), err)
		}
	}

	return nil
}

func (m *Manager) Mount(router *gin.Engine) {
	if router == nil {
		return
	}

	for _, module := range m.modules {
		module.Mount(router)
	}
}

func (m *Manager) NavigationItems() []navigation.Item {
	var items []navigation.Item
	for _, module := range m.modules {
		items = append(items, module.NavigationItems()...)
	}
	return items
}
