package sections

import (
	"fmt"
	"html/template"
	"strings"
	"sync"

	"constructor-script-backend/internal/models"
)

// RenderContext exposes the minimal capabilities required by section renderers.
type RenderContext interface {
	// SanitizeHTML should clean potentially unsafe markup before rendering.
	SanitizeHTML(input string) string
	// CloneTemplates returns an isolated template instance for rendering complex sections.
	CloneTemplates() (*template.Template, error)
}

// Renderer describes a function capable of rendering a section element into HTML output and optional scripts.
type Renderer func(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string)

// Registry stores the mapping between section element types and their renderers.
type Registry struct {
	mu        sync.RWMutex
	renderers map[string]Renderer
}

// NewRegistry creates an empty section renderer registry.
func NewRegistry() *Registry {
	return &Registry{renderers: make(map[string]Renderer)}
}

// Register associates a renderer with a normalised element type. It returns an error when the input is invalid.
func (r *Registry) Register(sectionType string, renderer Renderer) error {
	if r == nil {
		return fmt.Errorf("registry is nil")
	}

	sectionType = strings.TrimSpace(strings.ToLower(sectionType))
	if sectionType == "" {
		return fmt.Errorf("section type is empty")
	}
	if renderer == nil {
		return fmt.Errorf("renderer is nil for type %s", sectionType)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.renderers == nil {
		r.renderers = make(map[string]Renderer)
	}
	r.renderers[sectionType] = renderer
	return nil
}

// MustRegister registers the renderer and panics if registration fails.
func (r *Registry) MustRegister(sectionType string, renderer Renderer) {
	if err := r.Register(sectionType, renderer); err != nil {
		panic(err)
	}
}

// Get retrieves a renderer for the provided section type if it exists.
func (r *Registry) Get(sectionType string) (Renderer, bool) {
	if r == nil {
		return nil, false
	}

	sectionType = strings.TrimSpace(strings.ToLower(sectionType))
	if sectionType == "" {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	renderer, ok := r.renderers[sectionType]
	return renderer, ok
}

// Clone creates a copy of the registry with the same renderer mappings.
func (r *Registry) Clone() *Registry {
	if r == nil {
		return NewRegistry()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	cloned := NewRegistry()
	for key, renderer := range r.renderers {
		cloned.renderers[key] = renderer
	}
	return cloned
}
