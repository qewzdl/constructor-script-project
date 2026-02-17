package service

import (
	"testing"

	"constructor-script-backend/internal/models"
)

func TestGetPageBuilderConfig_IncludesSectionVariationSchemas(t *testing.T) {
	service := &PageService{}
	config := service.GetPageBuilderConfig()

	assertSectionVariation(t, config, "hero", "split", []string{"split", "centered", "minimal", "immersive"})
	assertSectionVariation(t, config, "features", "cards", []string{"cards", "list", "spotlight", "glyph", "constellation"})
	assertSectionVariation(t, config, "steps", "numbered", []string{"numbered"})
	assertSectionVariation(t, config, "catalog", "cards", []string{"cards", "compact", "highlighted"})
}

func assertSectionVariation(t *testing.T, config models.PageBuilderConfig, sectionType string, expectedDefault string, expectedValues []string) {
	t.Helper()

	sections := config.AvailableSections
	var section *models.SectionTypeConfig
	for i := range sections {
		if sections[i].Type == sectionType {
			section = &sections[i]
			break
		}
	}
	if section == nil {
		t.Fatalf("section %q is missing from page builder config", sectionType)
	}

	rawVariation, ok := section.Schema["variation"]
	if !ok {
		t.Fatalf("section %q is missing variation schema", sectionType)
	}
	variationSchema, ok := rawVariation.(map[string]interface{})
	if !ok {
		t.Fatalf("section %q has invalid variation schema type %T", sectionType, rawVariation)
	}

	defaultValue, _ := variationSchema["default"].(string)
	if defaultValue != expectedDefault {
		t.Fatalf("section %q default variation mismatch: got %q, want %q", sectionType, defaultValue, expectedDefault)
	}

	options := parseVariationOptions(variationSchema["options"])
	if len(options) != len(expectedValues) {
		t.Fatalf("section %q variation option count mismatch: got %d, want %d", sectionType, len(options), len(expectedValues))
	}
	for _, expected := range expectedValues {
		if !containsOption(options, expected) {
			t.Fatalf("section %q is missing variation option %q", sectionType, expected)
		}
	}
}

func parseVariationOptions(value interface{}) []string {
	switch typed := value.(type) {
	case []map[string]string:
		result := make([]string, 0, len(typed))
		for _, option := range typed {
			if candidate, ok := option["value"]; ok && candidate != "" {
				result = append(result, candidate)
			}
		}
		return result
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, raw := range typed {
			option, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			candidate, _ := option["value"].(string)
			if candidate != "" {
				result = append(result, candidate)
			}
		}
		return result
	default:
		return nil
	}
}

func containsOption(options []string, value string) bool {
	for _, option := range options {
		if option == value {
			return true
		}
	}
	return false
}
