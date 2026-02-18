package service

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/theme"
)

type PrepareSectionsOptions struct {
	NormaliseSpacing bool
}

func PrepareSections(sections []models.Section, manager *theme.Manager, opts PrepareSectionsOptions) (models.PostSections, error) {
	if len(sections) == 0 {
		return models.PostSections{}, nil
	}

	prepared := make(models.PostSections, 0, len(sections))
	sectionDefinitions := sectionDefinitionsFromManager(manager)
	elementDefinitions := elementDefinitionsFromManager(manager)

	for i, section := range sections {
		sectionType := strings.TrimSpace(strings.ToLower(section.Type))
		if sectionType == "" {
			sectionType = "standard"
		}

		definition, ok := sectionDefinitions[sectionType]
		if !ok {
			return nil, fmt.Errorf("section %d: unknown type '%s'", i, sectionType)
		}

		section.Settings = normaliseSectionSettings(section.Settings, definition.Settings)

		allowElements := true
		if definition.SupportsElements != nil {
			allowElements = *definition.SupportsElements
		}
		allowedElements := definition.AllowedElementSet()

		if allowElements {
			if len(section.Elements) > 0 {
				preparedElements, err := prepareSectionElements(section.Elements, elementDefinitions, allowedElements)
				if err != nil {
					return nil, fmt.Errorf("section %d: %w", i, err)
				}
				section.Elements = preparedElements
			}
		} else {
			section.Elements = nil
		}

		if limitSetting, ok := definition.Settings["limit"]; ok {
			if section.Limit <= 0 && section.Settings != nil {
				if rawLimit, exists := section.Settings["limit"]; exists {
					if parsedLimit, parsed := parseSectionSettingInt(rawLimit); parsed {
						section.Limit = parsedLimit
					}
				}
			}
			section.Limit = clampSectionLimit(section.Limit, limitSetting)
			if section.Settings == nil {
				section.Settings = make(map[string]interface{})
			}
			section.Settings["limit"] = section.Limit
		} else if sectionType == "posts_list" {
			section.Limit = clampSectionLimit(section.Limit, theme.SectionSettingDefinition{
				Default: intPtr(constants.DefaultPostListSectionLimit),
				Min:     intPtr(1),
				Max:     intPtr(constants.MaxPostListSectionLimit),
			})
		} else if sectionType == "categories_list" {
			section.Limit = clampSectionLimit(section.Limit, theme.SectionSettingDefinition{
				Default: intPtr(constants.DefaultCategoryListSectionLimit),
				Min:     intPtr(1),
				Max:     intPtr(constants.MaxCategoryListSectionLimit),
			})
		}

		if modeSetting, ok := definition.Settings["mode"]; ok {
			if strings.TrimSpace(section.Mode) == "" && section.Settings != nil {
				if rawMode, exists := section.Settings["mode"]; exists {
					section.Mode = strings.TrimSpace(fmt.Sprint(rawMode))
				}
			}
			section.Mode = normaliseSectionMode(section.Mode, modeSetting)
			if section.Settings == nil {
				section.Settings = make(map[string]interface{})
			}
			section.Settings["mode"] = section.Mode
		} else {
			section.Mode = strings.TrimSpace(strings.ToLower(section.Mode))
		}
		section.Variation = definition.NormaliseVariation(extractSectionVariation(section.Variation, section.Settings))

		if section.ID == "" {
			section.ID = uuid.New().String()
		}

		if section.Order == 0 {
			section.Order = i + 1
		}

		if opts.NormaliseSpacing {
			defaultPadding := constants.DefaultSectionPadding
			if manager != nil {
				if activeTheme := manager.Active(); activeTheme != nil {
					defaultPadding = activeTheme.DefaultSectionPadding()
				}
			}
			section.PaddingVertical = normaliseSectionPadding(section.PaddingVertical, defaultPadding)
			section.PaddingTop = normaliseSectionPaddingSide(section.PaddingTop)
			section.PaddingBottom = normaliseSectionPaddingSide(section.PaddingBottom)
			section.MarginVertical = normaliseSectionMargin(section.MarginVertical)
			section.MarginTop = normaliseSectionMarginSide(section.MarginTop)
			section.MarginBottom = normaliseSectionMarginSide(section.MarginBottom)
		}

		section.Type = sectionType

		prepared = append(prepared, section)
	}

	return prepared, nil
}

func prepareSectionElements(elements []models.SectionElement, definitions map[string]theme.ElementDefinition, allowed map[string]struct{}) ([]models.SectionElement, error) {
	prepared := make([]models.SectionElement, 0, len(elements))

	for i, elem := range elements {
		if elem.ID == "" {
			elem.ID = uuid.New().String()
		}

		if elem.Order == 0 {
			elem.Order = i + 1
		}

		elemType := strings.ToLower(strings.TrimSpace(elem.Type))
		if elemType == "" {
			return nil, fmt.Errorf("element %d: type is required", i)
		}
		if _, ok := definitions[elemType]; !ok {
			return nil, fmt.Errorf("element %d: unknown type '%s'", i, elem.Type)
		}
		if len(allowed) > 0 {
			if _, ok := allowed[elemType]; !ok {
				return nil, fmt.Errorf("element %d: type '%s' is not allowed in this section", i, elem.Type)
			}
		}
		elem.Type = elemType

		if elem.Content == nil {
			return nil, fmt.Errorf("element %d: content is required", i)
		}

		prepared = append(prepared, elem)
	}

	return prepared, nil
}

func sectionDefinitionsFromManager(manager *theme.Manager) map[string]theme.SectionDefinition {
	if manager == nil {
		return theme.DefaultSectionDefinitions()
	}
	if active := manager.Active(); active != nil {
		defs := active.SectionDefinitions()
		if len(defs) > 0 {
			return defs
		}
	}
	return theme.DefaultSectionDefinitions()
}

func elementDefinitionsFromManager(manager *theme.Manager) map[string]theme.ElementDefinition {
	if manager == nil {
		return theme.DefaultElementDefinitions()
	}
	if active := manager.Active(); active != nil {
		defs := active.ElementDefinitions()
		if len(defs) > 0 {
			return defs
		}
	}
	return theme.DefaultElementDefinitions()
}

func clampSectionLimit(value int, setting theme.SectionSettingDefinition) int {
	result := value
	if result <= 0 {
		if setting.Default != nil {
			result = *setting.Default
		} else {
			result = constants.DefaultPostListSectionLimit
		}
	}
	if setting.Min != nil && result < *setting.Min {
		result = *setting.Min
	}
	if setting.Max != nil && result > *setting.Max {
		result = *setting.Max
	} else if setting.Max == nil && result > constants.MaxPostListSectionLimit {
		result = constants.MaxPostListSectionLimit
	}
	if result <= 0 {
		result = 1
	}
	return result
}

func normaliseSectionMode(value string, setting theme.SectionSettingDefinition) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if len(setting.Options) == 0 {
		return trimmed
	}

	allowed := make(map[string]struct{}, len(setting.Options))
	first := ""
	for _, option := range setting.Options {
		optionValue := strings.TrimSpace(strings.ToLower(option.Value))
		if optionValue == "" {
			continue
		}
		if first == "" {
			first = optionValue
		}
		allowed[optionValue] = struct{}{}
	}

	if trimmed != "" {
		if _, ok := allowed[trimmed]; ok {
			return trimmed
		}
	}

	fallback := strings.TrimSpace(strings.ToLower(setting.DefaultValue))
	if fallback != "" {
		if _, ok := allowed[fallback]; ok {
			return fallback
		}
	}

	if first != "" {
		return first
	}

	return trimmed
}

func intPtr(value int) *int {
	return &value
}

func clampSectionPaddingValue(value int) int {
	options := constants.SectionPaddingOptions()
	if len(options) == 0 {
		return 0
	}
	if value <= options[0] {
		return options[0]
	}
	last := options[len(options)-1]
	if value >= last {
		return last
	}
	closest := options[0]
	minDiff := absInt(value - closest)
	for _, option := range options[1:] {
		diff := absInt(value - option)
		if diff < minDiff {
			closest = option
			minDiff = diff
		}
	}
	return closest
}

func normaliseSectionPadding(value *int, defaultPadding int) *int {
	if value == nil {
		return intPtr(defaultPadding)
	}
	normalised := clampSectionPaddingValue(*value)
	return intPtr(normalised)
}

func normaliseSectionPaddingSide(value *int) *int {
	if value == nil {
		return nil
	}
	normalised := clampSectionPaddingValue(*value)
	return intPtr(normalised)
}

func clampSectionMarginValue(value int) int {
	options := constants.SectionMarginOptions()
	if len(options) == 0 {
		return 0
	}
	if value <= options[0] {
		return options[0]
	}
	last := options[len(options)-1]
	if value >= last {
		return last
	}
	closest := options[0]
	minDiff := absInt(value - closest)
	for _, option := range options[1:] {
		diff := absInt(value - option)
		if diff < minDiff {
			closest = option
			minDiff = diff
		}
	}
	return closest
}

func normaliseSectionMargin(value *int) *int {
	if value == nil {
		return nil
	}
	normalised := clampSectionMarginValue(*value)
	return intPtr(normalised)
}

func normaliseSectionMarginSide(value *int) *int {
	if value == nil {
		return nil
	}
	normalised := clampSectionMarginValue(*value)
	return intPtr(normalised)
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func extractSectionVariation(value string, settings map[string]interface{}) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	if settings == nil {
		return ""
	}

	if raw, ok := settings["variation"].(string); ok {
		return strings.TrimSpace(raw)
	}
	if raw, ok := settings["Variation"].(string); ok {
		return strings.TrimSpace(raw)
	}

	return ""
}

func normaliseSectionSettings(
	source map[string]interface{},
	definitions map[string]theme.SectionSettingDefinition,
) map[string]interface{} {
	if len(source) == 0 && len(definitions) == 0 {
		return nil
	}

	cloned := cloneSettings(source)
	if len(definitions) == 0 {
		return cloned
	}

	index := make(map[string]interface{}, len(cloned))
	for key, value := range cloned {
		normalisedKey := normaliseSectionSettingKey(key)
		if normalisedKey == "" {
			continue
		}
		if _, exists := index[normalisedKey]; exists {
			continue
		}
		index[normalisedKey] = value
	}

	keys := make([]string, 0, len(definitions))
	for key := range definitions {
		normalisedKey := normaliseSectionSettingKey(key)
		if normalisedKey == "" {
			continue
		}
		keys = append(keys, normalisedKey)
	}
	sort.Strings(keys)

	result := make(map[string]interface{}, len(cloned)+len(keys))
	known := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		known[key] = struct{}{}
		definition, exists := definitions[key]
		if !exists {
			// Fall back to lookup by non-normalised key, when configuration keys differ in case.
			for definitionKey, candidate := range definitions {
				if normaliseSectionSettingKey(definitionKey) == key {
					definition = candidate
					exists = true
					break
				}
			}
		}
		if !exists {
			continue
		}

		rawValue, hasRaw := index[key]
		value, hasValue := normaliseSectionSettingValue(rawValue, hasRaw, definition)
		if hasValue {
			result[key] = value
		}
	}

	for key, value := range cloned {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		normalisedKey := normaliseSectionSettingKey(trimmedKey)
		if _, isKnown := known[normalisedKey]; isKnown {
			continue
		}
		if _, exists := result[trimmedKey]; exists {
			continue
		}
		result[trimmedKey] = cloneAny(value)
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func normaliseSectionSettingValue(
	raw interface{},
	hasRaw bool,
	definition theme.SectionSettingDefinition,
) (interface{}, bool) {
	fieldType := strings.TrimSpace(strings.ToLower(inferSettingType(definition)))
	if fieldType == "" {
		fieldType = "text"
	}

	switch fieldType {
	case "select":
		return normaliseSectionSelectValue(raw, hasRaw, definition)
	case "boolean":
		return normaliseSectionBooleanValue(raw, hasRaw, definition)
	case "number", "integer", "range":
		return normaliseSectionNumberValue(raw, hasRaw, definition)
	case "text", "string", "url", "textarea":
		return normaliseSectionTextValue(raw, hasRaw, definition)
	default:
		return normaliseSectionTextValue(raw, hasRaw, definition)
	}
}

func normaliseSectionSelectValue(
	raw interface{},
	hasRaw bool,
	definition theme.SectionSettingDefinition,
) (interface{}, bool) {
	if len(definition.Options) == 0 {
		return normaliseSectionTextValue(raw, hasRaw, definition)
	}

	first := ""
	options := make(map[string]string, len(definition.Options))
	for _, option := range definition.Options {
		value := strings.TrimSpace(option.Value)
		if value == "" {
			continue
		}
		normalised := strings.ToLower(value)
		if first == "" {
			first = value
		}
		if _, exists := options[normalised]; !exists {
			options[normalised] = value
		}
	}
	if len(options) == 0 {
		return nil, false
	}

	if hasRaw {
		candidate := strings.TrimSpace(fmt.Sprint(raw))
		if candidate != "" {
			if resolved, ok := options[strings.ToLower(candidate)]; ok {
				return resolved, true
			}
		}
	}

	defaultCandidate := strings.TrimSpace(definition.DefaultValue)
	if defaultCandidate != "" {
		if resolved, ok := options[strings.ToLower(defaultCandidate)]; ok {
			return resolved, true
		}
	}

	if first != "" {
		return first, true
	}

	return nil, false
}

func normaliseSectionBooleanValue(
	raw interface{},
	hasRaw bool,
	definition theme.SectionSettingDefinition,
) (interface{}, bool) {
	if hasRaw {
		if parsed, ok := parseSectionSettingBool(raw); ok {
			return parsed, true
		}
	}

	if parsed, ok := parseBoolDefault(definition.DefaultValue); ok {
		return parsed, true
	}

	if definition.Default != nil {
		return *definition.Default != 0, true
	}

	if definition.Required {
		return false, true
	}

	return nil, false
}

func normaliseSectionNumberValue(
	raw interface{},
	hasRaw bool,
	definition theme.SectionSettingDefinition,
) (interface{}, bool) {
	value := 0
	hasValue := false

	if hasRaw {
		if parsed, ok := parseSectionSettingInt(raw); ok {
			value = parsed
			hasValue = true
		}
	}

	if !hasValue && definition.Default != nil {
		value = *definition.Default
		hasValue = true
	}

	if !hasValue {
		if parsed, err := strconv.Atoi(strings.TrimSpace(definition.DefaultValue)); err == nil {
			value = parsed
			hasValue = true
		}
	}

	if !hasValue {
		if definition.Required {
			value = 0
			hasValue = true
		} else {
			return nil, false
		}
	}

	if definition.Min != nil && value < *definition.Min {
		value = *definition.Min
	}
	if definition.Max != nil && value > *definition.Max {
		value = *definition.Max
	}

	return value, true
}

func normaliseSectionTextValue(
	raw interface{},
	hasRaw bool,
	definition theme.SectionSettingDefinition,
) (interface{}, bool) {
	value := ""
	if hasRaw {
		value = strings.TrimSpace(fmt.Sprint(raw))
	}

	if value == "" {
		value = strings.TrimSpace(definition.DefaultValue)
	}

	if value == "" && definition.Required {
		return "", true
	}

	if value == "" {
		return nil, false
	}

	return value, true
}

func parseSectionSettingBool(raw interface{}) (bool, bool) {
	switch value := raw.(type) {
	case bool:
		return value, true
	case string:
		return parseBoolDefault(value)
	case int:
		return value != 0, true
	case int8:
		return value != 0, true
	case int16:
		return value != 0, true
	case int32:
		return value != 0, true
	case int64:
		return value != 0, true
	case uint:
		return value != 0, true
	case uint8:
		return value != 0, true
	case uint16:
		return value != 0, true
	case uint32:
		return value != 0, true
	case uint64:
		return value != 0, true
	case float32:
		return value != 0, true
	case float64:
		return value != 0, true
	default:
		return false, false
	}
}

func parseSectionSettingInt(raw interface{}) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int8:
		return int(value), true
	case int16:
		return int(value), true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case uint:
		return int(value), true
	case uint8:
		return int(value), true
	case uint16:
		return int(value), true
	case uint32:
		return int(value), true
	case uint64:
		return int(value), true
	case float32:
		return int(value), true
	case float64:
		return int(value), true
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func normaliseSectionSettingKey(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(trimmed))
	lastUnderscore := false
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}

	return strings.Trim(builder.String(), "_")
}

// NormaliseSections ensures section fields are consistently populated and ordered without requiring theme definitions.
func NormaliseSections(sections models.PostSections) models.PostSections {
	if len(sections) == 0 {
		return sections
	}

	normalised := make(models.PostSections, 0, len(sections))

	for i, section := range sections {
		section.Type = strings.TrimSpace(strings.ToLower(section.Type))
		if section.Type == "" {
			section.Type = "standard"
		}
		section.Variation = strings.TrimSpace(strings.ToLower(extractSectionVariation(section.Variation, section.Settings)))
		section.Title = strings.TrimSpace(section.Title)
		section.Description = strings.TrimSpace(section.Description)

		if section.ID == "" {
			section.ID = uuid.New().String()
		}
		if section.Order == 0 {
			section.Order = i + 1
		}

		section.PaddingVertical = normaliseSectionPadding(section.PaddingVertical, constants.DefaultSectionPadding)
		section.PaddingTop = normaliseSectionPaddingSide(section.PaddingTop)
		section.PaddingBottom = normaliseSectionPaddingSide(section.PaddingBottom)
		section.MarginVertical = normaliseSectionMargin(section.MarginVertical)
		section.MarginTop = normaliseSectionMarginSide(section.MarginTop)
		section.MarginBottom = normaliseSectionMarginSide(section.MarginBottom)

		if len(section.Elements) > 0 {
			elements := make([]models.SectionElement, 0, len(section.Elements))
			for j, element := range section.Elements {
				element.Type = strings.TrimSpace(strings.ToLower(element.Type))
				if element.ID == "" {
					element.ID = uuid.New().String()
				}
				if element.Order == 0 {
					element.Order = j + 1
				}
				elements = append(elements, element)
			}
			sort.SliceStable(elements, func(a, b int) bool {
				return elements[a].Order < elements[b].Order
			})
			section.Elements = elements
		}

		normalised = append(normalised, section)
	}

	sort.SliceStable(normalised, func(i, j int) bool {
		return normalised[i].Order < normalised[j].Order
	})

	return normalised
}
