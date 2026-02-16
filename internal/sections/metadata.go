package sections

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// SectionMetadata describes a section type with its configuration schema and display properties.
type SectionMetadata struct {
	Type                string                 `json:"type"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	Category            string                 `json:"category"`
	Icon                string                 `json:"icon,omitempty"`
	Schema              map[string]interface{} `json:"schema,omitempty"`
	Preview             string                 `json:"preview,omitempty"`
	Order               int                    `json:"order,omitempty"`
	AllowedIn           []string               `json:"allowed_in,omitempty"`
	AllowedElements     []string               `json:"allowed_elements,omitempty"`
	SupportsElements    *bool                  `json:"supports_elements,omitempty"`
	SupportsHeaderImage *bool                  `json:"supports_header_image,omitempty"`
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

	normalisedMetadata := normaliseSectionMetadata(desc.Metadata)
	sectionType := normalisedMetadata.Type
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
	r.descriptors[sectionType] = &SectionDescriptor{
		Renderer: desc.Renderer,
		Metadata: normalisedMetadata,
		Validate: desc.Validate,
	}
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
	return cloneSectionMetadata(desc.Metadata), true
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
		result = append(result, cloneSectionMetadata(desc.Metadata))
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Order == result[j].Order {
			return result[i].Type < result[j].Type
		}
		return result[i].Order < result[j].Order
	})
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
		cloned.descriptors[key] = cloneSectionDescriptor(desc)
	}

	return cloned
}

func normaliseSectionMetadata(metadata SectionMetadata) SectionMetadata {
	result := cloneSectionMetadata(metadata)
	result.Type = strings.TrimSpace(strings.ToLower(result.Type))
	result.Name = strings.TrimSpace(result.Name)
	result.Description = strings.TrimSpace(result.Description)
	result.Category = strings.TrimSpace(result.Category)
	result.Icon = strings.TrimSpace(result.Icon)
	result.Preview = strings.TrimSpace(result.Preview)
	result.AllowedIn = normaliseStringList(result.AllowedIn)
	result.AllowedElements = normaliseStringList(result.AllowedElements)
	if result.Order < 0 {
		result.Order = 0
	}
	return result
}

func cloneSectionDescriptor(desc *SectionDescriptor) *SectionDescriptor {
	if desc == nil {
		return nil
	}

	return &SectionDescriptor{
		Renderer: desc.Renderer,
		Metadata: cloneSectionMetadata(desc.Metadata),
		Validate: desc.Validate,
	}
}

func cloneSectionMetadata(metadata SectionMetadata) SectionMetadata {
	result := metadata
	result.Schema = cloneSchema(metadata.Schema)
	result.AllowedIn = cloneStringSlice(metadata.AllowedIn)
	result.AllowedElements = cloneStringSlice(metadata.AllowedElements)
	result.SupportsElements = cloneBoolPointer(metadata.SupportsElements)
	result.SupportsHeaderImage = cloneBoolPointer(metadata.SupportsHeaderImage)
	return result
}

func cloneSchema(source map[string]interface{}) map[string]interface{} {
	if len(source) == 0 {
		return nil
	}

	payload, err := json.Marshal(source)
	if err != nil {
		result := make(map[string]interface{}, len(source))
		for key, value := range source {
			result[key] = value
		}
		return result
	}

	var result map[string]interface{}
	if err := json.Unmarshal(payload, &result); err != nil {
		result = make(map[string]interface{}, len(source))
		for key, value := range source {
			result[key] = value
		}
	}
	return result
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, len(values))
	copy(result, values)
	return result
}

func cloneBoolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func normaliseStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalised := strings.TrimSpace(strings.ToLower(value))
		if normalised == "" {
			continue
		}
		if _, exists := seen[normalised]; exists {
			continue
		}
		seen[normalised] = struct{}{}
		result = append(result, normalised)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
