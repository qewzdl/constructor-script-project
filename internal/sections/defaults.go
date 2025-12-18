package sections

// DefaultRegistry returns a registry pre-populated with the built-in section renderers.
func DefaultRegistry() *Registry {
	reg := NewRegistry()
	RegisterDefaults(reg)
	return reg
}

// DefaultRegistryWithMetadata returns a registry with metadata support and all built-in sections.
func DefaultRegistryWithMetadata() *RegistryWithMetadata {
	reg := NewRegistryWithMetadata()
	RegisterDefaultsWithMetadata(reg)
	return reg
}

// RegisterDefaults adds the built-in section renderers to the provided registry.
// Registration errors are silently ignored to prevent panics in production.
func RegisterDefaults(reg *Registry) {
	if reg == nil {
		return
	}

	// Basic content sections
	RegisterParagraph(reg)
	RegisterImage(reg)
	RegisterImageGroup(reg)
	RegisterFileGroup(reg)
	RegisterList(reg)
	RegisterSearch(reg)
	RegisterFeatures(reg)
	RegisterHero(reg)

	// Profile sections
	RegisterProfileAccount(reg)
	RegisterProfileSecurity(reg)
	RegisterProfileCourses(reg)

	// Dynamic list sections
	RegisterPostsList(reg)
	RegisterCategoriesList(reg)
	RegisterCoursesList(reg)
}

// RegisterDefaultsWithMetadata adds all built-in sections with full metadata support.
func RegisterDefaultsWithMetadata(reg *RegistryWithMetadata) {
	if reg == nil {
		return
	}

	// For now, register basic sections without metadata
	// TODO: Convert all sections to use metadata descriptors
	RegisterParagraph(reg.Registry)
	RegisterImage(reg.Registry)
	RegisterImageGroup(reg.Registry)
	RegisterFileGroup(reg.Registry)
	RegisterList(reg.Registry)
	RegisterSearch(reg.Registry)
	RegisterProfileAccount(reg.Registry)
	RegisterProfileSecurity(reg.Registry)
	RegisterProfileCourses(reg.Registry)

	// Register dynamic sections with full metadata
	RegisterPostsListWithMetadata(reg)
	RegisterCategoriesListWithMetadata(reg)
	RegisterCoursesListWithMetadata(reg)
	RegisterHeroWithMetadata(reg)
	RegisterFeaturesWithMetadata(reg)
}
