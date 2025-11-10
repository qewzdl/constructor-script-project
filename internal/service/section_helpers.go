package service

import (
	"fmt"
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

		allowElements := true
		if definition.SupportsElements != nil {
			allowElements = *definition.SupportsElements
		}

		if allowElements {
			if len(section.Elements) > 0 {
				preparedElements, err := prepareSectionElements(section.Elements, elementDefinitions)
				if err != nil {
					return nil, fmt.Errorf("section %d: %w", i, err)
				}
				section.Elements = preparedElements
			}
		} else {
			section.Elements = nil
		}

		if limitSetting, ok := definition.Settings["limit"]; ok {
			section.Limit = clampSectionLimit(section.Limit, limitSetting)
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
			section.Mode = normaliseSectionMode(section.Mode, modeSetting)
		} else {
			section.Mode = strings.TrimSpace(strings.ToLower(section.Mode))
		}

		if section.ID == "" {
			section.ID = uuid.New().String()
		}

		if section.Order == 0 {
			section.Order = i + 1
		}

		if opts.NormaliseSpacing {
			section.PaddingVertical = normaliseSectionPadding(section.PaddingVertical)
			section.MarginVertical = normaliseSectionMargin(section.MarginVertical)
		}

		section.Type = sectionType

		prepared = append(prepared, section)
	}

	return prepared, nil
}

func prepareSectionElements(elements []models.SectionElement, definitions map[string]theme.ElementDefinition) ([]models.SectionElement, error) {
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

func normaliseSectionPadding(value *int) *int {
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

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
