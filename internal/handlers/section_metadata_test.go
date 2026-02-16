package handlers

import (
	"testing"

	internalsections "constructor-script-backend/internal/sections"
	"constructor-script-backend/internal/theme"
)

func TestMergeSectionMetadataDefinitions_AppliesExtendedFields(t *testing.T) {
	supportsElements := true
	supportsHeaderImage := true

	metadata := []internalsections.SectionMetadata{
		{
			Type:                "plugin_cards",
			Name:                "Plugin cards",
			Description:         "Plugin-defined section",
			Category:            "custom",
			Icon:                "sparkles",
			Preview:             "/static/previews/plugin-cards.png",
			Order:               42,
			AllowedIn:           []string{"page", "homepage", "PAGE"},
			AllowedElements:     []string{"paragraph", "image", "Paragraph"},
			SupportsElements:    &supportsElements,
			SupportsHeaderImage: &supportsHeaderImage,
			Schema: map[string]interface{}{
				"layout": map[string]interface{}{
					"type":    "string",
					"enum":    []string{"grid", "carousel"},
					"default": "grid",
				},
				"limit": map[string]interface{}{
					"type":    "number",
					"default": 6,
					"min":     1,
					"max":     12,
				},
			},
		},
	}

	merged := mergeSectionMetadataDefinitions(map[string]theme.SectionDefinition{}, metadata)
	definition, ok := merged["plugin_cards"]
	if !ok {
		t.Fatalf("expected plugin_cards definition to be merged")
	}

	if definition.Order != 42 {
		t.Fatalf("expected order to be 42, got %d", definition.Order)
	}
	if !sliceContains(definition.AllowedIn, "page") || !sliceContains(definition.AllowedIn, "homepage") {
		t.Fatalf("expected allowed_in to contain page and homepage, got %#v", definition.AllowedIn)
	}
	if !sliceContains(definition.AllowedElements, "paragraph") || !sliceContains(definition.AllowedElements, "image") {
		t.Fatalf("expected allowed_elements to contain paragraph and image, got %#v", definition.AllowedElements)
	}
	if definition.SupportsElements == nil || !*definition.SupportsElements {
		t.Fatalf("expected supports_elements=true")
	}
	if definition.SupportsHeaderImage == nil || !*definition.SupportsHeaderImage {
		t.Fatalf("expected supports_header_image=true")
	}

	layout, exists := definition.Settings["layout"]
	if !exists {
		t.Fatalf("expected layout setting to be present")
	}
	if len(layout.Options) != 2 {
		t.Fatalf("expected layout enum to be converted into two options, got %d", len(layout.Options))
	}
	if layout.DefaultValue != "grid" {
		t.Fatalf("expected layout default value to be 'grid', got %q", layout.DefaultValue)
	}

	limit, exists := definition.Settings["limit"]
	if !exists {
		t.Fatalf("expected limit setting to be present")
	}
	if limit.Default == nil || *limit.Default != 6 {
		t.Fatalf("expected limit default to be 6, got %#v", limit.Default)
	}
	if limit.Min == nil || *limit.Min != 1 {
		t.Fatalf("expected limit min to be 1, got %#v", limit.Min)
	}
	if limit.Max == nil || *limit.Max != 12 {
		t.Fatalf("expected limit max to be 12, got %#v", limit.Max)
	}
}

func TestMetadataSchemaFieldToSetting_HandlesEnumAndSnakeCaseFlags(t *testing.T) {
	setting, ok := metadataSchemaFieldToSetting(map[string]interface{}{
		"type":                "string",
		"enum":                []interface{}{"cards", "compact"},
		"default":             "cards",
		"allow_anchor_picker": true,
		"allow_media_browse":  "true",
	})
	if !ok {
		t.Fatalf("expected schema field to convert into setting definition")
	}

	if len(setting.Options) != 2 {
		t.Fatalf("expected 2 options from enum, got %d", len(setting.Options))
	}
	if setting.DefaultValue != "cards" {
		t.Fatalf("expected default value 'cards', got %q", setting.DefaultValue)
	}
	if !setting.AllowAnchorPicker {
		t.Fatalf("expected allow_anchor_picker=true")
	}
	if !setting.AllowMediaBrowse {
		t.Fatalf("expected allow_media_browse=true")
	}
}

func sliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
