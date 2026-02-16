package service

import (
	"sort"
	"strings"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/theme"
)

// SectionCatalogOptions controls which optional section groups are exposed.
type SectionCatalogOptions struct {
	BlogEnabled    bool
	CoursesEnabled bool
}

// SectionCatalog provides a single source of truth for builder-facing section metadata.
type SectionCatalog struct {
	sections map[string]theme.SectionDefinition
	elements map[string]theme.ElementDefinition
}

// SectionCatalog returns a catalog derived from the active theme definitions.
func (s *PageService) SectionCatalog(options SectionCatalogOptions) *SectionCatalog {
	var manager *theme.Manager
	if s != nil {
		manager = s.themes
	}
	return NewSectionCatalog(manager, options)
}

// BuilderSectionDefinitions returns normalised section definitions for builder UIs.
func (s *PageService) BuilderSectionDefinitions(options SectionCatalogOptions) map[string]theme.SectionDefinition {
	return s.SectionCatalog(options).SectionDefinitions()
}

// BuilderElementDefinitions returns normalised element definitions for builder UIs.
func (s *PageService) BuilderElementDefinitions(options SectionCatalogOptions) map[string]theme.ElementDefinition {
	return s.SectionCatalog(options).ElementDefinitions()
}

// NewSectionCatalog builds a catalog from theme definitions and optional feature flags.
func NewSectionCatalog(manager *theme.Manager, options SectionCatalogOptions) *SectionCatalog {
	sections := cloneSectionDefinitions(sectionDefinitionsFromManager(manager))
	elements := cloneElementDefinitions(elementDefinitionsFromManager(manager))

	if !options.BlogEnabled {
		delete(sections, "posts_list")
		delete(sections, "categories_list")
	}

	if !options.CoursesEnabled {
		delete(sections, "courses_list")
		delete(sections, "catalog")
	}

	return &SectionCatalog{
		sections: sections,
		elements: elements,
	}
}

// SectionDefinitions returns a defensive copy of section definitions.
func (c *SectionCatalog) SectionDefinitions() map[string]theme.SectionDefinition {
	if c == nil {
		return nil
	}
	return cloneSectionDefinitions(c.sections)
}

// ElementDefinitions returns a defensive copy of element definitions.
func (c *SectionCatalog) ElementDefinitions() map[string]theme.ElementDefinition {
	if c == nil {
		return nil
	}
	return cloneElementDefinitions(c.elements)
}

// SectionTypeConfigs converts section + element definitions to builder API payload format.
func (c *SectionCatalog) SectionTypeConfigs() []models.SectionTypeConfig {
	if c == nil {
		return nil
	}

	sectionDefs := c.sortedSectionDefinitions()
	elementDefs := c.sortedElementDefinitions()

	configs := make([]models.SectionTypeConfig, 0, len(sectionDefs)+len(elementDefs))

	for _, sectionDef := range sectionDefs {
		config := models.SectionTypeConfig{
			Type:        sectionDef.Type,
			Name:        firstNonEmpty(strings.TrimSpace(sectionDef.Label), displayNameFromType(sectionDef.Type)),
			Description: strings.TrimSpace(sectionDef.Description),
			Category:    firstNonEmpty(strings.TrimSpace(sectionDef.Category), defaultSectionCategory(sectionDef.Type)),
			Icon:        firstNonEmpty(strings.TrimSpace(sectionDef.Icon), defaultSectionIcon(sectionDef.Type)),
			Preview:     strings.TrimSpace(sectionDef.Preview),
			AllowedIn:   resolveAllowedIn(sectionDef.AllowedIn, defaultSectionAllowedIn(sectionDef.Type)),
			Schema:      buildSectionSchema(sectionDef),
		}
		if len(config.AllowedIn) == 0 {
			config.AllowedIn = nil
		}
		if len(config.Schema) == 0 {
			config.Schema = nil
		}
		configs = append(configs, config)
	}

	for _, elementDef := range elementDefs {
		allowedIn := resolveAllowedIn(elementDef.AllowedIn, elementAllowedIn(elementDef.Type, sectionDefs))
		config := models.SectionTypeConfig{
			Type:        elementDef.Type,
			Name:        firstNonEmpty(strings.TrimSpace(elementDef.Label), displayNameFromType(elementDef.Type)),
			Description: strings.TrimSpace(elementDef.Description),
			Category:    firstNonEmpty(strings.TrimSpace(elementDef.Category), defaultElementCategory(elementDef.Type)),
			Icon:        firstNonEmpty(strings.TrimSpace(elementDef.Icon), defaultElementIcon(elementDef.Type)),
			Preview:     strings.TrimSpace(elementDef.Preview),
			AllowedIn:   allowedIn,
		}
		if len(config.AllowedIn) == 0 {
			config.AllowedIn = nil
		}
		configs = append(configs, config)
	}

	return configs
}

func (c *SectionCatalog) sortedSectionDefinitions() []theme.SectionDefinition {
	if c == nil || len(c.sections) == 0 {
		return nil
	}

	result := make([]theme.SectionDefinition, 0, len(c.sections))
	for _, definition := range c.sections {
		definition.Type = normaliseType(definition.Type)
		if definition.Type == "" {
			continue
		}
		result = append(result, definition)
	}

	sort.SliceStable(result, func(i, j int) bool {
		left := result[i]
		right := result[j]
		if left.Order == right.Order {
			return left.Type < right.Type
		}
		return left.Order < right.Order
	})

	return result
}

func (c *SectionCatalog) sortedElementDefinitions() []theme.ElementDefinition {
	if c == nil || len(c.elements) == 0 {
		return nil
	}

	result := make([]theme.ElementDefinition, 0, len(c.elements))
	for _, definition := range c.elements {
		definition.Type = normaliseType(definition.Type)
		if definition.Type == "" {
			continue
		}
		result = append(result, definition)
	}

	sort.SliceStable(result, func(i, j int) bool {
		left := result[i]
		right := result[j]
		if left.Order == right.Order {
			return left.Type < right.Type
		}
		return left.Order < right.Order
	})

	return result
}

func buildSectionSchema(definition theme.SectionDefinition) map[string]interface{} {
	if len(definition.Settings) == 0 && len(definition.Variations) == 0 {
		return nil
	}

	result := make(map[string]interface{})

	if len(definition.Settings) > 0 {
		keys := make([]string, 0, len(definition.Settings))
		for key := range definition.Settings {
			trimmed := strings.TrimSpace(key)
			if trimmed == "" {
				continue
			}
			keys = append(keys, trimmed)
		}
		sort.Strings(keys)

		for _, key := range keys {
			result[key] = buildSettingSchema(key, definition.Settings[key])
		}
	}

	if _, exists := result["variation"]; !exists {
		if variationField := buildVariationSchema(definition); len(variationField) > 0 {
			result["variation"] = variationField
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func buildSettingSchema(key string, definition theme.SectionSettingDefinition) map[string]interface{} {
	fieldType := inferSettingType(definition)

	result := map[string]interface{}{
		"type": fieldType,
	}

	label := strings.TrimSpace(definition.Label)
	if label == "" {
		label = displayNameFromType(key)
	}
	if label != "" {
		result["label"] = label
	}

	if placeholder := strings.TrimSpace(definition.Placeholder); placeholder != "" {
		result["placeholder"] = placeholder
	}

	if definition.Min != nil {
		result["min"] = *definition.Min
	}
	if definition.Max != nil {
		result["max"] = *definition.Max
	}
	if definition.Default != nil {
		result["default"] = *definition.Default
	}

	if len(definition.Options) > 0 {
		options := make([]map[string]string, 0, len(definition.Options))
		for _, option := range definition.Options {
			value := strings.TrimSpace(option.Value)
			if value == "" {
				continue
			}
			labelValue := strings.TrimSpace(option.Label)
			if labelValue == "" {
				labelValue = value
			}
			options = append(options, map[string]string{
				"value": value,
				"label": labelValue,
			})
		}
		if len(options) > 0 {
			result["options"] = options
		}
	}

	if defaultValue := strings.TrimSpace(definition.DefaultValue); defaultValue != "" {
		if _, hasDefault := result["default"]; !hasDefault {
			if fieldType == "boolean" {
				if parsed, ok := parseBoolDefault(defaultValue); ok {
					result["default"] = parsed
				} else {
					result["default"] = defaultValue
				}
			} else {
				result["default"] = defaultValue
			}
		}
		result["defaultValue"] = defaultValue
	}

	if perPageLabel := strings.TrimSpace(definition.PerPageLabel); perPageLabel != "" {
		result["perPageLabel"] = perPageLabel
	}

	if definition.Required {
		result["required"] = true
	}
	if definition.AllowMediaBrowse {
		result["allowMediaBrowse"] = true
	}
	if definition.AllowAnchorPicker {
		result["allowAnchorPicker"] = true
	}
	if definition.AllowCoursePicker {
		result["allowCoursePicker"] = true
	}
	if definition.AllowPostPicker {
		result["allowPostPicker"] = true
	}
	if len(definition.HiddenForVariations) > 0 {
		result["hiddenForVariations"] = cloneStringSlice(
			definition.HiddenForVariations,
		)
	}

	return result
}

func buildVariationSchema(definition theme.SectionDefinition) map[string]interface{} {
	if len(definition.Variations) == 0 {
		return nil
	}

	options := make([]map[string]string, 0, len(definition.Variations))
	for _, variation := range definition.Variations {
		value := normaliseType(variation.Value)
		if value == "" {
			continue
		}

		label := strings.TrimSpace(variation.Label)
		if label == "" {
			label = displayNameFromType(value)
		}

		options = append(options, map[string]string{
			"value": value,
			"label": label,
		})
	}

	if len(options) == 0 {
		return nil
	}

	result := map[string]interface{}{
		"type":    "select",
		"label":   "Variation",
		"options": options,
	}

	if defaultValue := definition.DefaultVariation(); defaultValue != "" {
		result["default"] = defaultValue
	}

	return result
}

func inferSettingType(definition theme.SectionSettingDefinition) string {
	if explicit := strings.TrimSpace(strings.ToLower(definition.Type)); explicit != "" {
		return explicit
	}
	if len(definition.Options) > 0 {
		return "select"
	}
	if definition.Min != nil || definition.Max != nil || definition.Default != nil {
		return "number"
	}
	if parsed, ok := parseBoolDefault(definition.DefaultValue); ok {
		_ = parsed
		return "boolean"
	}
	return "text"
}

func parseBoolDefault(value string) (bool, bool) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func elementAllowedIn(elementType string, sectionDefs []theme.SectionDefinition) []string {
	elementType = normaliseType(elementType)
	if elementType == "" || len(sectionDefs) == 0 {
		return nil
	}

	result := make([]string, 0, len(sectionDefs))
	for _, sectionDef := range sectionDefs {
		supportsElements := true
		if sectionDef.SupportsElements != nil {
			supportsElements = *sectionDef.SupportsElements
		}
		if !supportsElements {
			continue
		}

		allowedSet := sectionDef.AllowedElementSet()
		if len(allowedSet) == 0 {
			result = append(result, sectionDef.Type)
			continue
		}
		if _, ok := allowedSet[elementType]; ok {
			result = append(result, sectionDef.Type)
		}
	}

	return normaliseContextList(result)
}

func resolveAllowedIn(primary []string, fallback []string) []string {
	allowed := normaliseContextList(primary)
	if len(allowed) > 0 {
		return allowed
	}
	return normaliseContextList(fallback)
}

func normaliseContextList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		normalised := normaliseType(value)
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

func defaultSectionAllowedIn(sectionType string) []string {
	switch normaliseType(sectionType) {
	case "standard", "grid", "features", "file_list":
		return []string{"page", "post", "homepage"}
	default:
		return []string{"page", "homepage"}
	}
}

func defaultSectionCategory(sectionType string) string {
	switch normaliseType(sectionType) {
	case "standard", "grid", "file_list":
		return "layout"
	case "hero", "features":
		return "marketing"
	case "posts_list", "courses_list", "catalog":
		return "content"
	case "categories_list":
		return "navigation"
	case "contact":
		return "support"
	default:
		return "custom"
	}
}

func defaultSectionIcon(sectionType string) string {
	switch normaliseType(sectionType) {
	case "standard":
		return "layout"
	case "hero":
		return "star"
	case "features":
		return "sparkles"
	case "grid", "catalog":
		return "grid"
	case "contact":
		return "phone"
	case "posts_list":
		return "list"
	case "categories_list":
		return "tag"
	case "courses_list":
		return "book"
	case "file_list":
		return "folder"
	default:
		return "layers"
	}
}

func defaultElementCategory(elementType string) string {
	switch normaliseType(elementType) {
	case "image", "image_group", "file_group":
		return "media"
	case "search":
		return "interactive"
	default:
		return "elements"
	}
}

func defaultElementIcon(elementType string) string {
	switch normaliseType(elementType) {
	case "paragraph":
		return "type"
	case "image":
		return "image"
	case "image_group":
		return "images"
	case "file_group":
		return "file-text"
	case "list":
		return "list"
	case "search":
		return "search"
	case "feature_item":
		return "sparkles"
	case "profile_account_details":
		return "user"
	case "profile_security":
		return "shield"
	default:
		return "box"
	}
}

func displayNameFromType(value string) string {
	normalised := normaliseType(value)
	if normalised == "" {
		return ""
	}

	parts := strings.Split(normalised, "_")
	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		parts[i] = strings.ToUpper(trimmed[:1]) + trimmed[1:]
	}

	return strings.Join(parts, " ")
}

func firstNonEmpty(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func normaliseType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func cloneSectionDefinitions(definitions map[string]theme.SectionDefinition) map[string]theme.SectionDefinition {
	if len(definitions) == 0 {
		return map[string]theme.SectionDefinition{}
	}

	result := make(map[string]theme.SectionDefinition, len(definitions))
	for key, definition := range definitions {
		typeKey := normaliseType(definition.Type)
		if typeKey == "" {
			typeKey = normaliseType(key)
		}
		if typeKey == "" {
			continue
		}
		definition.Type = typeKey
		definition.AllowedElements = cloneStringSlice(definition.AllowedElements)
		definition.AllowedIn = cloneStringSlice(definition.AllowedIn)
		definition.Variations = cloneSectionVariations(definition.Variations)
		definition.Settings = cloneSectionSettings(definition.Settings)
		result[typeKey] = definition
	}
	return result
}

func cloneElementDefinitions(definitions map[string]theme.ElementDefinition) map[string]theme.ElementDefinition {
	if len(definitions) == 0 {
		return map[string]theme.ElementDefinition{}
	}

	result := make(map[string]theme.ElementDefinition, len(definitions))
	for key, definition := range definitions {
		typeKey := normaliseType(definition.Type)
		if typeKey == "" {
			typeKey = normaliseType(key)
		}
		if typeKey == "" {
			continue
		}
		definition.Type = typeKey
		definition.AllowedIn = cloneStringSlice(definition.AllowedIn)
		result[typeKey] = definition
	}
	return result
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	clone := make([]string, len(values))
	copy(clone, values)
	return clone
}

func cloneSectionVariations(values []theme.SectionVariationDefinition) []theme.SectionVariationDefinition {
	if len(values) == 0 {
		return nil
	}
	clone := make([]theme.SectionVariationDefinition, len(values))
	copy(clone, values)
	return clone
}

func cloneSectionSettings(values map[string]theme.SectionSettingDefinition) map[string]theme.SectionSettingDefinition {
	if len(values) == 0 {
		return nil
	}

	result := make(map[string]theme.SectionSettingDefinition, len(values))
	for key, value := range values {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		result[trimmed] = cloneSectionSetting(value)
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func cloneSectionSetting(value theme.SectionSettingDefinition) theme.SectionSettingDefinition {
	clone := value
	clone.Min = cloneIntPointer(value.Min)
	clone.Max = cloneIntPointer(value.Max)
	clone.Default = cloneIntPointer(value.Default)

	if len(value.Options) > 0 {
		options := make([]theme.SectionSettingOption, len(value.Options))
		copy(options, value.Options)
		clone.Options = options
	} else {
		clone.Options = nil
	}
	clone.HiddenForVariations = cloneStringSlice(value.HiddenForVariations)

	return clone
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
