package handlers

import "constructor-script-backend/internal/sections"

// RegisterSectionRenderer allows runtime registration of custom section renderers.
func (h *TemplateHandler) RegisterSectionRenderer(sectionType string, renderer sections.Renderer) error {
	if h == nil {
		return nil
	}

	if h.sectionRegistry == nil {
		h.sectionRegistry = sections.DefaultRegistry()
	}

	return h.sectionRegistry.Register(sectionType, renderer)
}

// RegisterSectionWithMetadata allows plugins to register sections with full metadata support.
func (h *TemplateHandler) RegisterSectionWithMetadata(desc *sections.SectionDescriptor) error {
	if h == nil {
		return nil
	}

	// Convert current registry to one with metadata support if needed
	metaReg, ok := h.sectionRegistry.(*sections.RegistryWithMetadata)
	if !ok {
		// Create new metadata registry
		newReg := sections.NewRegistryWithMetadata()

		// Copy existing renderers if we had a basic registry
		if basicReg, isBasic := h.sectionRegistry.(*sections.Registry); isBasic {
			newReg.Registry = basicReg.Clone()
		}

		h.sectionRegistry = newReg
		metaReg = newReg
	}

	return metaReg.RegisterWithMetadata(desc)
}

// SectionRegistry exposes a copy of the internal registry for inspection or extension.
func (h *TemplateHandler) SectionRegistry() *sections.Registry {
	if h == nil {
		return sections.NewRegistry()
	}

	if metaReg, ok := h.sectionRegistry.(*sections.RegistryWithMetadata); ok {
		return metaReg.Registry.Clone()
	}

	if reg, ok := h.sectionRegistry.(*sections.Registry); ok {
		return reg.Clone()
	}

	return sections.NewRegistry()
}

// SectionMetadata returns metadata for all registered sections if available.
func (h *TemplateHandler) SectionMetadata() []sections.SectionMetadata {
	if h == nil {
		return nil
	}

	if metaReg, ok := h.sectionRegistry.(*sections.RegistryWithMetadata); ok {
		return metaReg.ListMetadata()
	}

	return nil
}
