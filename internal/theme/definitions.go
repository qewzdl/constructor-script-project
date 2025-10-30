package theme

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	defaults := defaultSectionDefinitions()

	filePath := filepath.Join(themePath, "data", "admin", "sections.json")
	content, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaults, nil
		}
		return nil, fmt.Errorf("read section definitions: %w", err)
	}

	var file sectionDefinitionFile
	if err := json.Unmarshal(content, &file); err != nil {
		return nil, fmt.Errorf("parse section definitions: %w", err)
	}

	if len(file.Types) == 0 {
		return defaults, nil
	}

	result := make(map[string]SectionDefinition, len(file.Types))
	for _, def := range file.Types {
		normalised := strings.ToLower(strings.TrimSpace(def.Type))
		if normalised == "" {
			continue
		}

		def.Type = normalised

		if existing, ok := defaults[normalised]; ok {
			def = mergeSectionDefinition(existing, def)
		} else {
			def = mergeSectionDefinition(SectionDefinition{Type: normalised}, def)
		}

		result[normalised] = def
	}

	// Ensure defaults are always available.
	for key, def := range defaults {
		if _, ok := result[key]; !ok {
			result[key] = def
		}
	}

	return result, nil
}

func loadElementDefinitions(themePath string) (map[string]ElementDefinition, error) {
	defaults := defaultElementDefinitions()

	filePath := filepath.Join(themePath, "data", "admin", "elements.json")
	content, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaults, nil
		}
		return nil, fmt.Errorf("read element definitions: %w", err)
	}

	var file elementDefinitionFile
	if err := json.Unmarshal(content, &file); err != nil {
		return nil, fmt.Errorf("parse element definitions: %w", err)
	}

	if len(file.Types) == 0 {
		return defaults, nil
	}

	result := make(map[string]ElementDefinition, len(file.Types))
	for _, def := range file.Types {
		normalised := strings.ToLower(strings.TrimSpace(def.Type))
		if normalised == "" {
			continue
		}
		def.Type = normalised
		if existing, ok := defaults[normalised]; ok {
			result[normalised] = mergeElementDefinition(existing, def)
		} else {
			result[normalised] = def
		}
	}

	for key, def := range defaults {
		if _, ok := result[key]; !ok {
			result[key] = def
		}
	}

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

	limitDefault := 6
	limitMin := 1
	limitMax := 24

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
