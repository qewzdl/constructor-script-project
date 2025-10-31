package handlers

import "constructor-script-backend/internal/sections"

// RegisterSectionRenderer allows runtime registration of custom section renderers.
func (h *TemplateHandler) RegisterSectionRenderer(sectionType string, renderer sections.Renderer) error {
	if h == nil {
		return nil
	}

	if h.sectionRegistry == nil {
		h.sectionRegistry = sections.NewRegistry()
	}

	return h.sectionRegistry.Register(sectionType, renderer)
}

// SectionRegistry exposes a copy of the internal registry for inspection or extension.
func (h *TemplateHandler) SectionRegistry() *sections.Registry {
	if h == nil {
		return sections.NewRegistry()
	}
	return h.sectionRegistry.Clone()
}
