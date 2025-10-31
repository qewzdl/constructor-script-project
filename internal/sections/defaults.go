package sections

// DefaultRegistry returns a registry pre-populated with the built-in section renderers.
func DefaultRegistry() *Registry {
	reg := NewRegistry()
	RegisterDefaults(reg)
	return reg
}

// RegisterDefaults adds the built-in section renderers to the provided registry.
func RegisterDefaults(reg *Registry) {
	if reg == nil {
		return
	}

	RegisterParagraph(reg)
	RegisterImage(reg)
	RegisterImageGroup(reg)
	RegisterList(reg)
	RegisterSearch(reg)
}
