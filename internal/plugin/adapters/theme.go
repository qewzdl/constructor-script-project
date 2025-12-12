package adapters

import (
	"constructor-script-backend/internal/theme"
	"constructor-script-backend/pkg/pluginsdk"
)

// ThemeManagerAdapter adapts theme manager to pluginsdk.ThemeManager interface
type ThemeManagerAdapter struct {
	manager *theme.Manager
}

func NewThemeManagerAdapter(manager *theme.Manager) pluginsdk.ThemeManager {
	return &ThemeManagerAdapter{manager: manager}
}

func (a *ThemeManagerAdapter) RenderTemplate(name string, data interface{}) (string, error) {
	// Theme rendering is typically done through template handlers
	// This is a placeholder that should be implemented based on actual requirements
	return "", nil
}

func (a *ThemeManagerAdapter) GetActiveTheme() string {
	theme := a.manager.Active()
	if theme == nil {
		return ""
	}
	return theme.Slug
}

func (a *ThemeManagerAdapter) ThemeExists(name string) bool {
	_, exists := a.manager.Resolve(name)
	return exists
}
