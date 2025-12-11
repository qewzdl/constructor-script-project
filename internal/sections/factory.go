package sections

import (
	"fmt"
	"sync"
)

// SectionFactory manages the creation and registration of section types.
// It provides a centralized way to define and instantiate sections.
type SectionFactory struct {
	mu         sync.RWMutex
	blueprints map[string]*SectionBlueprint
	registry   *RegistryWithMetadata
}

// SectionBlueprint defines a template for creating section instances.
type SectionBlueprint struct {
	Type        string
	Name        string
	Description string
	Category    string
	Icon        string
	Builder     func() *SectionBuilder
}

// NewSectionFactory creates a new factory with an empty registry.
func NewSectionFactory() *SectionFactory {
	return &SectionFactory{
		blueprints: make(map[string]*SectionBlueprint),
		registry:   NewRegistryWithMetadata(),
	}
}

// NewSectionFactoryWithRegistry creates a factory using an existing registry.
func NewSectionFactoryWithRegistry(registry *RegistryWithMetadata) *SectionFactory {
	if registry == nil {
		registry = NewRegistryWithMetadata()
	}
	return &SectionFactory{
		blueprints: make(map[string]*SectionBlueprint),
		registry:   registry,
	}
}

// RegisterBlueprint registers a section blueprint for later instantiation.
func (f *SectionFactory) RegisterBlueprint(blueprint *SectionBlueprint) error {
	if f == nil {
		return fmt.Errorf("factory is nil")
	}
	if blueprint == nil {
		return fmt.Errorf("blueprint is nil")
	}
	if blueprint.Type == "" {
		return fmt.Errorf("blueprint type is required")
	}
	if blueprint.Builder == nil {
		return fmt.Errorf("blueprint builder is required")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.blueprints[blueprint.Type] = blueprint

	// Build and register the section
	builder := blueprint.Builder()
	desc, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build section from blueprint: %w", err)
	}

	return f.registry.RegisterWithMetadata(desc)
}

// CreateSection builds a section descriptor from a registered blueprint.
func (f *SectionFactory) CreateSection(sectionType string) (*SectionDescriptor, error) {
	if f == nil {
		return nil, fmt.Errorf("factory is nil")
	}

	f.mu.RLock()
	blueprint, ok := f.blueprints[sectionType]
	f.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no blueprint registered for type: %s", sectionType)
	}

	builder := blueprint.Builder()
	return builder.Build()
}

// GetRegistry returns the underlying registry.
func (f *SectionFactory) GetRegistry() *RegistryWithMetadata {
	if f == nil {
		return nil
	}
	return f.registry
}

// ListBlueprints returns all registered blueprints.
func (f *SectionFactory) ListBlueprints() []*SectionBlueprint {
	if f == nil {
		return nil
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	blueprints := make([]*SectionBlueprint, 0, len(f.blueprints))
	for _, bp := range f.blueprints {
		blueprints = append(blueprints, bp)
	}
	return blueprints
}

// HasBlueprint checks if a blueprint exists for the given type.
func (f *SectionFactory) HasBlueprint(sectionType string) bool {
	if f == nil {
		return false
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	_, ok := f.blueprints[sectionType]
	return ok
}

// Clone creates a copy of the factory with the same blueprints.
func (f *SectionFactory) Clone() *SectionFactory {
	if f == nil {
		return NewSectionFactory()
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	cloned := &SectionFactory{
		blueprints: make(map[string]*SectionBlueprint),
		registry:   f.registry.CloneWithMetadata(),
	}

	for key, bp := range f.blueprints {
		cloned.blueprints[key] = bp
	}

	return cloned
}
