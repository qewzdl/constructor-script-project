package sections

// DefaultRegistry returns a registry pre-populated with the built-in section renderers.
func DefaultRegistry() *Registry {
	reg := NewRegistry()
	RegisterDefaults(reg)
	return reg
}

// RegisterDefaults adds the built-in section renderers to the provided registry.
// Registration errors are silently ignored to prevent panics in production.
func RegisterDefaults(reg *Registry) {
	if reg == nil {
		return
	}

	RegisterParagraph(reg)
	RegisterImage(reg)
	RegisterImageGroup(reg)
	RegisterFileGroup(reg)
	RegisterList(reg)
	RegisterSearch(reg)
	RegisterProfileAccount(reg)
	RegisterProfileSecurity(reg)
	RegisterProfileCourses(reg)
}
