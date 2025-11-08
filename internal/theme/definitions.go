package theme

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"constructor-script-backend/internal/constants"
)

// SectionDefinition describes a section type that can be used by the content builder
// and validated by the backend.
type SectionDefinition struct {
	Type             string                              `json:"type"`
	Label            string                              `json:"label,omitempty"`
	Order            int                                 `json:"order,omitempty"`
	Description      string                              `json:"description,omitempty"`
	SupportsElements *bool                               `json:"supports_elements,omitempty"`
	Settings         map[string]SectionSettingDefinition `json:"settings,omitempty"`
}

// SectionSettingDefinition describes additional configuration for a section type.
type SectionSettingDefinition struct {
	Label   string `json:"label,omitempty"`
	Min     *int   `json:"min,omitempty"`
	Max     *int   `json:"max,omitempty"`
	Default *int   `json:"default,omitempty"`
}

// ElementDefinition represents a single element type definition that can be
// referenced in sections.
type ElementDefinition struct {
	Type        string `json:"type"`
	Label       string `json:"label,omitempty"`
	Order       int    `json:"order,omitempty"`
	Description string `json:"description,omitempty"`
}

type sectionDefinitionFile struct {
	Types []SectionDefinition `json:"types"`
}

type elementDefinitionFile struct {
	Types []ElementDefinition `json:"types"`
}

func loadDefinitions(themePath string) (map[string]SectionDefinition, map[string]ElementDefinition, error) {
	sections, err := loadSectionDefinitions(themePath)
	if err != nil {
		return nil, nil, err
	}

	elements, err := loadElementDefinitions(themePath)
	if err != nil {
		return nil, nil, err
	}

	return sections, elements, nil
}

func loadSectionDefinitions(themePath string) (map[string]SectionDefinition, error) {
	result := defaultSectionDefinitions()

	filePath := filepath.Join(themePath, "data", "admin", "sections.json")
	definitions, err := readSectionDefinitionFile(filePath)
	if err != nil {
		return nil, err
	}
	applySectionDefinitions(result, definitions)

	directoryPath := filepath.Join(themePath, "data", "admin", "sections")
	definitions, err = readSectionDefinitionDirectory(directoryPath)
	if err != nil {
		return nil, err
	}
	applySectionDefinitions(result, definitions)

	return result, nil
}

func loadElementDefinitions(themePath string) (map[string]ElementDefinition, error) {
	result := defaultElementDefinitions()

	filePath := filepath.Join(themePath, "data", "admin", "elements.json")
	definitions, err := readElementDefinitionFile(filePath)
	if err != nil {
		return nil, err
	}
	applyElementDefinitions(result, definitions)

	directoryPath := filepath.Join(themePath, "data", "admin", "elements")
	definitions, err = readElementDefinitionDirectory(directoryPath)
	if err != nil {
		return nil, err
	}
	applyElementDefinitions(result, definitions)

	return result, nil
}

func mergeSectionDefinition(base, override SectionDefinition) SectionDefinition {
	result := base

	if override.Label != "" {
		result.Label = override.Label
	}
	if override.Order != 0 {
		result.Order = override.Order
	}
	if override.Description != "" {
		result.Description = override.Description
	}
	if override.SupportsElements != nil {
		result.SupportsElements = override.SupportsElements
	}

	if len(override.Settings) > 0 {
		if result.Settings == nil {
			result.Settings = make(map[string]SectionSettingDefinition, len(override.Settings))
		}
		for key, setting := range override.Settings {
			result.Settings[key] = mergeSectionSetting(result.Settings[key], setting)
		}
	}

	return result
}

func mergeSectionSetting(base, override SectionSettingDefinition) SectionSettingDefinition {
	result := base

	if override.Label != "" {
		result.Label = override.Label
	}
	if override.Min != nil {
		result.Min = override.Min
	}
	if override.Max != nil {
		result.Max = override.Max
	}
	if override.Default != nil {
		result.Default = override.Default
	}

	return result
}

func readSectionDefinitionFile(filePath string) ([]SectionDefinition, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read section definitions: %w", err)
	}

	definitions, err := parseSectionDefinitions(content)
	if err != nil {
		return nil, fmt.Errorf("parse section definitions: %w", err)
	}
	return definitions, nil
}

func readSectionDefinitionDirectory(dirPath string) ([]SectionDefinition, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("list section definitions: %w", err)
	}

	var definitions []SectionDefinition
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read section definition %s: %w", entry.Name(), err)
		}

		parsed, err := parseSectionDefinitions(content)
		if err != nil {
			return nil, fmt.Errorf("parse section definition %s: %w", entry.Name(), err)
		}

		if len(parsed) > 0 {
			definitions = append(definitions, parsed...)
		}
	}

	return definitions, nil
}

func parseSectionDefinitions(content []byte) ([]SectionDefinition, error) {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return nil, nil
	}

	switch trimmed[0] {
	case '{':
		var file sectionDefinitionFile
		if err := json.Unmarshal(trimmed, &file); err == nil {
			if len(file.Types) > 0 {
				return file.Types, nil
			}
		}

		var single SectionDefinition
		if err := json.Unmarshal(trimmed, &single); err == nil {
			if strings.TrimSpace(single.Type) == "" {
				return nil, nil
			}
			return []SectionDefinition{single}, nil
		}
		return nil, fmt.Errorf("unsupported section definition format")
	case '[':
		var list []SectionDefinition
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return nil, fmt.Errorf("parse section definition array: %w", err)
		}
		return list, nil
	default:
		return nil, fmt.Errorf("invalid section definition content")
	}
}

func applySectionDefinitions(target map[string]SectionDefinition, definitions []SectionDefinition) {
	if len(definitions) == 0 {
		return
	}

	for _, definition := range definitions {
		normalised := strings.ToLower(strings.TrimSpace(definition.Type))
		if normalised == "" {
			continue
		}

		definition.Type = normalised
		base, ok := target[normalised]
		if !ok {
			base = SectionDefinition{Type: normalised}
		}

		target[normalised] = mergeSectionDefinition(base, definition)
	}
}

func readElementDefinitionFile(filePath string) ([]ElementDefinition, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read element definitions: %w", err)
	}

	definitions, err := parseElementDefinitions(content)
	if err != nil {
		return nil, fmt.Errorf("parse element definitions: %w", err)
	}
	return definitions, nil
}

func readElementDefinitionDirectory(dirPath string) ([]ElementDefinition, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("list element definitions: %w", err)
	}

	var definitions []ElementDefinition
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read element definition %s: %w", entry.Name(), err)
		}

		parsed, err := parseElementDefinitions(content)
		if err != nil {
			return nil, fmt.Errorf("parse element definition %s: %w", entry.Name(), err)
		}

		if len(parsed) > 0 {
			definitions = append(definitions, parsed...)
		}
	}

	return definitions, nil
}

func parseElementDefinitions(content []byte) ([]ElementDefinition, error) {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return nil, nil
	}

	switch trimmed[0] {
	case '{':
		var file elementDefinitionFile
		if err := json.Unmarshal(trimmed, &file); err == nil {
			if len(file.Types) > 0 {
				return file.Types, nil
			}
		}

		var single ElementDefinition
		if err := json.Unmarshal(trimmed, &single); err == nil {
			if strings.TrimSpace(single.Type) == "" {
				return nil, nil
			}
			return []ElementDefinition{single}, nil
		}
		return nil, fmt.Errorf("unsupported element definition format")
	case '[':
		var list []ElementDefinition
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return nil, fmt.Errorf("parse element definition array: %w", err)
		}
		return list, nil
	default:
		return nil, fmt.Errorf("invalid element definition content")
	}
}

func applyElementDefinitions(target map[string]ElementDefinition, definitions []ElementDefinition) {
	if len(definitions) == 0 {
		return
	}

	for _, definition := range definitions {
		normalised := strings.ToLower(strings.TrimSpace(definition.Type))
		if normalised == "" {
			continue
		}

		definition.Type = normalised

		base, ok := target[normalised]
		if !ok {
			base = ElementDefinition{Type: normalised}
		}

		target[normalised] = mergeElementDefinition(base, definition)
	}
}

func mergeElementDefinition(base, override ElementDefinition) ElementDefinition {
	result := base

	if override.Label != "" {
		result.Label = override.Label
	}
	if override.Order != 0 {
		result.Order = override.Order
	}
	if override.Description != "" {
		result.Description = override.Description
	}

	return result
}

func defaultSectionDefinitions() map[string]SectionDefinition {
	standardSupports := true
	heroSupports := false
	postsSupports := false
	categoriesSupports := false
	coursesSupports := false

	limitDefault := 6
	limitMin := 1
	limitMax := 24

	categoryLimitDefault := constants.DefaultCategoryListSectionLimit
	categoryLimitMin := 1
	categoryLimitMax := constants.MaxCategoryListSectionLimit

	courseLimitDefault := constants.DefaultCourseListSectionLimit
	courseLimitMin := 1
	courseLimitMax := constants.MaxCourseListSectionLimit

	return map[string]SectionDefinition{
		"standard": {
			Type:             "standard",
			Label:            "Standard section",
			Order:            0,
			Description:      "Flexible content area for combining paragraphs, media, and lists.",
			SupportsElements: &standardSupports,
		},
		"hero": {
			Type:             "hero",
			Label:            "Hero section",
			Order:            10,
			Description:      "Prominent introduction block without additional content elements.",
			SupportsElements: &heroSupports,
		},
		"grid": {
			Type:             "grid",
			Label:            "Grid section",
			Order:            15,
			Description:      "Displays content blocks in a responsive grid layout.",
			SupportsElements: &standardSupports,
		},
		"posts_list": {
			Type:             "posts_list",
			Label:            "Posts list",
			Order:            20,
			Description:      "Automatically displays the most recent blog posts.",
			SupportsElements: &postsSupports,
			Settings: map[string]SectionSettingDefinition{
				"limit": {
					Label:   "Number of posts to display",
					Default: &limitDefault,
					Min:     &limitMin,
					Max:     &limitMax,
				},
			},
		},
		"categories_list": {
			Type:             "categories_list",
			Label:            "Categories list",
			Order:            18,
			Description:      "Displays a list of blog categories for quick topic navigation.",
			SupportsElements: &categoriesSupports,
			Settings: map[string]SectionSettingDefinition{
				"limit": {
					Label:   "Number of categories to display",
					Default: &categoryLimitDefault,
					Min:     &categoryLimitMin,
					Max:     &categoryLimitMax,
				},
			},
		},
		"courses_list": {
			Type:             "courses_list",
			Label:            "Courses list",
			Order:            22,
			Description:      "Highlights available course packages with pricing and topics.",
			SupportsElements: &coursesSupports,
			Settings: map[string]SectionSettingDefinition{
				"limit": {
					Label:   "Number of courses to display",
					Default: &courseLimitDefault,
					Min:     &courseLimitMin,
					Max:     &courseLimitMax,
				},
			},
		},
	}
}

func defaultElementDefinitions() map[string]ElementDefinition {
	return map[string]ElementDefinition{
		"paragraph": {
			Type:        "paragraph",
			Label:       "Paragraph",
			Order:       10,
			Description: "Text block for paragraphs and rich content.",
		},
		"image": {
			Type:        "image",
			Label:       "Image",
			Order:       20,
			Description: "Single image with optional alt text and caption.",
		},
		"image_group": {
			Type:        "image_group",
			Label:       "Image group",
			Order:       30,
			Description: "Collection of related images displayed together.",
		},
		"list": {
			Type:        "list",
			Label:       "List",
			Order:       40,
			Description: "Bulleted or numbered list of key points.",
		},
		"search": {
			Type:        "search",
			Label:       "Search",
			Order:       50,
			Description: "Embeds a search form within the section.",
		},
		"profile_account_details": {
			Type:        "profile_account_details",
			Label:       "Account details form",
			Order:       60,
			Description: "Profile form for updating username, email, and role.",
		},
		"profile_security": {
			Type:        "profile_security",
			Label:       "Security form",
			Order:       70,
			Description: "Password update form for the profile page.",
		},
		"profile_courses": {
			Type:        "profile_courses",
			Label:       "Courses list",
			Order:       80,
			Description: "Displays the learner's current course access with enrollment details.",
		},
	}
}

// DefaultSectionDefinitions returns a copy of the built-in section definitions.
func DefaultSectionDefinitions() map[string]SectionDefinition {
	defs := defaultSectionDefinitions()
	clone := make(map[string]SectionDefinition, len(defs))
	for key, value := range defs {
		clone[key] = value
	}
	return clone
}

// DefaultElementDefinitions returns a copy of the built-in element definitions.
func DefaultElementDefinitions() map[string]ElementDefinition {
	defs := defaultElementDefinitions()
	clone := make(map[string]ElementDefinition, len(defs))
	for key, value := range defs {
		clone[key] = value
	}
	return clone
}
