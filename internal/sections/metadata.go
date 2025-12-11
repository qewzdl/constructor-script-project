package sections

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// SectionMetadata describes a section type with its configuration schema and display properties.
type SectionMetadata struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Icon        string                 `json:"icon,omitempty"`
	Schema      map[string]interface{} `json:"schema,omitempty"`
	Preview     string                 `json:"preview,omitempty"`
}

// MetadataProvider returns metadata for a section type.
type MetadataProvider func() SectionMetadata

// Validator validates section element data before rendering.
type Validator func(elem interface{}) error

// SectionDescriptor wraps a renderer with its metadata and optional validator.
type SectionDescriptor struct {
	Renderer Renderer
	Metadata SectionMetadata
	Validate Validator
}

// RegistryWithMetadata extends Registry with metadata capabilities.
type RegistryWithMetadata struct {
	*Registry
	mu          sync.RWMutex
	descriptors map[string]*SectionDescriptor
}

// NewRegistryWithMetadata creates a registry that supports metadata.
func NewRegistryWithMetadata() *RegistryWithMetadata {
	return &RegistryWithMetadata{
		Registry:    NewRegistry(),
		descriptors: make(map[string]*SectionDescriptor),
	}
}

// RegisterWithMetadata registers a section with full metadata support.
func (r *RegistryWithMetadata) RegisterWithMetadata(desc *SectionDescriptor) error {
	if r == nil {
		return fmt.Errorf("registry is nil")
	}
	if desc == nil {
		return fmt.Errorf("descriptor is nil")
	}

	sectionType := strings.TrimSpace(strings.ToLower(desc.Metadata.Type))
	if sectionType == "" {
		return fmt.Errorf("section type is empty")
	}
	if desc.Renderer == nil {
		return fmt.Errorf("renderer is nil for type %s", sectionType)
	}

	// Register the renderer
	if err := r.Registry.Register(sectionType, desc.Renderer); err != nil {
		return err
	}

	// Store the full descriptor
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.descriptors == nil {
		r.descriptors = make(map[string]*SectionDescriptor)
	}
	r.descriptors[sectionType] = desc
	return nil
}

// GetMetadata retrieves metadata for a section type.
func (r *RegistryWithMetadata) GetMetadata(sectionType string) (SectionMetadata, bool) {
	if r == nil {
		return SectionMetadata{}, false
	}

	sectionType = strings.TrimSpace(strings.ToLower(sectionType))
	r.mu.RLock()
	defer r.mu.RUnlock()

	desc, ok := r.descriptors[sectionType]
	if !ok {
		return SectionMetadata{}, false
	}
	return desc.Metadata, true
}

// GetValidator retrieves validator for a section type if one exists.
func (r *RegistryWithMetadata) GetValidator(sectionType string) (Validator, bool) {
	if r == nil {
		return nil, false
	}

	sectionType = strings.TrimSpace(strings.ToLower(sectionType))
	r.mu.RLock()
	defer r.mu.RUnlock()

	desc, ok := r.descriptors[sectionType]
	if !ok || desc.Validate == nil {
		return nil, false
	}
	return desc.Validate, true
}

// ListMetadata returns metadata for all registered sections.
func (r *RegistryWithMetadata) ListMetadata() []SectionMetadata {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]SectionMetadata, 0, len(r.descriptors))
	for _, desc := range r.descriptors {
		result = append(result, desc.Metadata)
	}
	return result
}

// MarshalMetadataJSON returns JSON representation of all section metadata.
func (r *RegistryWithMetadata) MarshalMetadataJSON() ([]byte, error) {
	metadata := r.ListMetadata()
	return json.Marshal(metadata)
}

// CloneWithMetadata creates a copy including metadata.
func (r *RegistryWithMetadata) CloneWithMetadata() *RegistryWithMetadata {
	if r == nil {
		return NewRegistryWithMetadata()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	cloned := NewRegistryWithMetadata()
	cloned.Registry = r.Registry.Clone()

	for key, desc := range r.descriptors {
		cloned.descriptors[key] = desc
	}

	return cloned
}
