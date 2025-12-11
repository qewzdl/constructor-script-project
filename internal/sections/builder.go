package sections

import (
	"encoding/json"
	"fmt"
)

// SectionBuilder provides a fluent interface for creating section descriptors.
type SectionBuilder struct {
	descriptor *SectionDescriptor
	errors     []error
}

// NewSectionBuilder creates a new builder for section descriptors.
func NewSectionBuilder(sectionType string) *SectionBuilder {
	return &SectionBuilder{
		descriptor: &SectionDescriptor{
			Metadata: SectionMetadata{
				Type:   sectionType,
				Schema: make(map[string]interface{}),
			},
		},
	}
}

// WithName sets the display name of the section.
func (b *SectionBuilder) WithName(name string) *SectionBuilder {
	b.descriptor.Metadata.Name = name
	return b
}

// WithDescription sets the description of the section.
func (b *SectionBuilder) WithDescription(desc string) *SectionBuilder {
	b.descriptor.Metadata.Description = desc
	return b
}

// WithCategory sets the category for grouping sections.
func (b *SectionBuilder) WithCategory(category string) *SectionBuilder {
	b.descriptor.Metadata.Category = category
	return b
}

// WithIcon sets the icon identifier for the section.
func (b *SectionBuilder) WithIcon(icon string) *SectionBuilder {
	b.descriptor.Metadata.Icon = icon
	return b
}

// WithPreview sets a preview image or template for the section.
func (b *SectionBuilder) WithPreview(preview string) *SectionBuilder {
	b.descriptor.Metadata.Preview = preview
	return b
}

// WithRenderer sets the rendering function for the section.
func (b *SectionBuilder) WithRenderer(renderer Renderer) *SectionBuilder {
	if renderer == nil {
		b.errors = append(b.errors, fmt.Errorf("renderer cannot be nil"))
	}
	b.descriptor.Renderer = renderer
	return b
}

// WithValidator sets an optional validation function.
func (b *SectionBuilder) WithValidator(validator Validator) *SectionBuilder {
	b.descriptor.Validate = validator
	return b
}

// AddSchemaField adds a field definition to the section's schema.
func (b *SectionBuilder) AddSchemaField(name string, fieldSchema map[string]interface{}) *SectionBuilder {
	if b.descriptor.Metadata.Schema == nil {
		b.descriptor.Metadata.Schema = make(map[string]interface{})
	}
	b.descriptor.Metadata.Schema[name] = fieldSchema
	return b
}

// AddStringField is a convenience method for adding a string field.
func (b *SectionBuilder) AddStringField(name string, required bool, defaultValue ...string) *SectionBuilder {
	field := map[string]interface{}{
		"type": "string",
	}
	if required {
		field["required"] = true
	}
	if len(defaultValue) > 0 {
		field["default"] = defaultValue[0]
	}
	return b.AddSchemaField(name, field)
}

// AddNumberField is a convenience method for adding a number field.
func (b *SectionBuilder) AddNumberField(name string, min, max int, defaultValue ...int) *SectionBuilder {
	field := map[string]interface{}{
		"type": "number",
		"min":  min,
		"max":  max,
	}
	if len(defaultValue) > 0 {
		field["default"] = defaultValue[0]
	}
	return b.AddSchemaField(name, field)
}

// AddBooleanField is a convenience method for adding a boolean field.
func (b *SectionBuilder) AddBooleanField(name string, defaultValue bool) *SectionBuilder {
	field := map[string]interface{}{
		"type":    "boolean",
		"default": defaultValue,
	}
	return b.AddSchemaField(name, field)
}

// AddEnumField is a convenience method for adding an enum field.
func (b *SectionBuilder) AddEnumField(name string, options []string, defaultValue ...string) *SectionBuilder {
	field := map[string]interface{}{
		"type": "string",
		"enum": options,
	}
	if len(defaultValue) > 0 {
		field["default"] = defaultValue[0]
	}
	return b.AddSchemaField(name, field)
}

// AddArrayField is a convenience method for adding an array field.
func (b *SectionBuilder) AddArrayField(name, itemType string, minItems, maxItems int) *SectionBuilder {
	field := map[string]interface{}{
		"type": "array",
		"items": map[string]interface{}{
			"type": itemType,
		},
	}
	if minItems > 0 {
		field["minItems"] = minItems
	}
	if maxItems > 0 {
		field["maxItems"] = maxItems
	}
	return b.AddSchemaField(name, field)
}

// WithSchema sets the entire schema at once (useful for complex schemas).
func (b *SectionBuilder) WithSchema(schema map[string]interface{}) *SectionBuilder {
	b.descriptor.Metadata.Schema = schema
	return b
}

// WithSchemaFromJSON sets the schema from a JSON string.
func (b *SectionBuilder) WithSchemaFromJSON(jsonSchema string) *SectionBuilder {
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(jsonSchema), &schema); err != nil {
		b.errors = append(b.errors, fmt.Errorf("failed to parse schema JSON: %w", err))
	} else {
		b.descriptor.Metadata.Schema = schema
	}
	return b
}

// Build constructs the final SectionDescriptor and returns any accumulated errors.
func (b *SectionBuilder) Build() (*SectionDescriptor, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("builder has %d error(s): %v", len(b.errors), b.errors[0])
	}

	if b.descriptor.Renderer == nil {
		return nil, fmt.Errorf("renderer is required")
	}

	if b.descriptor.Metadata.Type == "" {
		return nil, fmt.Errorf("section type is required")
	}

	return b.descriptor, nil
}

// MustBuild builds the descriptor and panics if there are errors.
// Use this only when you're certain the configuration is valid.
func (b *SectionBuilder) MustBuild() *SectionDescriptor {
	desc, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build section descriptor: %v", err))
	}
	return desc
}
