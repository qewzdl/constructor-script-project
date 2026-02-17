package service

import (
	"testing"

	"constructor-script-backend/internal/models"
)

func TestSectionCatalogFiltersByFeatureFlags(t *testing.T) {
	catalog := NewSectionCatalog(nil, SectionCatalogOptions{
		BlogEnabled:    false,
		CoursesEnabled: false,
	})

	definitions := catalog.SectionDefinitions()

	if _, exists := definitions["posts_list"]; exists {
		t.Fatalf("posts_list should be hidden when blog feature is disabled")
	}
	if _, exists := definitions["categories_list"]; exists {
		t.Fatalf("categories_list should be hidden when blog feature is disabled")
	}
	if _, exists := definitions["courses_list"]; exists {
		t.Fatalf("courses_list should be hidden when courses feature is disabled")
	}
	if _, exists := definitions["catalog"]; exists {
		t.Fatalf("catalog should be hidden when courses feature is disabled")
	}
	if _, exists := definitions["standard"]; !exists {
		t.Fatalf("standard section should remain available")
	}
}

func TestSectionCatalogBuildsVariationSchema(t *testing.T) {
	catalog := NewSectionCatalog(nil, SectionCatalogOptions{
		BlogEnabled:    true,
		CoursesEnabled: true,
	})

	configs := catalog.SectionTypeConfigs()
	hero, ok := findSectionTypeConfig(configs, "hero")
	if !ok {
		t.Fatalf("hero section config is missing")
	}

	variationRaw, hasVariation := hero.Schema["variation"]
	if !hasVariation {
		t.Fatalf("hero variation schema is missing")
	}

	variation, ok := variationRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("hero variation schema has invalid type: %T", variationRaw)
	}

	defaultValue, _ := variation["default"].(string)
	if defaultValue != "split" {
		t.Fatalf("unexpected hero variation default: got %q, want %q", defaultValue, "split")
	}

	options := parseSchemaOptionValues(variation["options"])
	if !containsString(options, "split") || !containsString(options, "centered") || !containsString(options, "minimal") || !containsString(options, "immersive") {
		t.Fatalf("hero variation options are incomplete: %v", options)
	}

	imageURLRaw, hasImageURL := hero.Schema["image_url"]
	if !hasImageURL {
		t.Fatalf("hero image_url schema is missing")
	}
	imageURL, ok := imageURLRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("hero image_url schema has invalid type: %T", imageURLRaw)
	}
	hiddenVariations := parseSchemaStringValues(imageURL["hiddenForVariations"])
	if !containsString(hiddenVariations, "immersive") {
		t.Fatalf("hero image_url schema should be hidden for immersive variation, got %v", hiddenVariations)
	}

	imageAltRaw, hasImageAlt := hero.Schema["image_alt"]
	if !hasImageAlt {
		t.Fatalf("hero image_alt schema is missing")
	}
	imageAlt, ok := imageAltRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("hero image_alt schema has invalid type: %T", imageAltRaw)
	}
	hiddenAltVariations := parseSchemaStringValues(imageAlt["hiddenForVariations"])
	if !containsString(hiddenAltVariations, "immersive") {
		t.Fatalf("hero image_alt schema should be hidden for immersive variation, got %v", hiddenAltVariations)
	}

	primaryButtonIconRaw, hasPrimaryButtonIcon := hero.Schema["button_icon"]
	if !hasPrimaryButtonIcon {
		t.Fatalf("hero button_icon schema is missing")
	}
	primaryButtonIcon, ok := primaryButtonIconRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("hero button_icon schema has invalid type: %T", primaryButtonIconRaw)
	}
	hiddenPrimaryButtonIconVariations := parseSchemaStringValues(primaryButtonIcon["hiddenForVariations"])
	if !containsString(hiddenPrimaryButtonIconVariations, "split") ||
		!containsString(hiddenPrimaryButtonIconVariations, "centered") ||
		!containsString(hiddenPrimaryButtonIconVariations, "minimal") {
		t.Fatalf("hero button_icon schema should be hidden for split/centered/minimal variations, got %v", hiddenPrimaryButtonIconVariations)
	}

	secondaryButtonIconRaw, hasSecondaryButtonIcon := hero.Schema["secondary_button_icon"]
	if !hasSecondaryButtonIcon {
		t.Fatalf("hero secondary_button_icon schema is missing")
	}
	secondaryButtonIcon, ok := secondaryButtonIconRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("hero secondary_button_icon schema has invalid type: %T", secondaryButtonIconRaw)
	}
	hiddenSecondaryButtonIconVariations := parseSchemaStringValues(secondaryButtonIcon["hiddenForVariations"])
	if !containsString(hiddenSecondaryButtonIconVariations, "split") ||
		!containsString(hiddenSecondaryButtonIconVariations, "centered") ||
		!containsString(hiddenSecondaryButtonIconVariations, "minimal") {
		t.Fatalf("hero secondary_button_icon schema should be hidden for split/centered/minimal variations, got %v", hiddenSecondaryButtonIconVariations)
	}
}

func TestSectionCatalogDerivesElementAllowedIn(t *testing.T) {
	catalog := NewSectionCatalog(nil, SectionCatalogOptions{
		BlogEnabled:    true,
		CoursesEnabled: true,
	})

	configs := catalog.SectionTypeConfigs()
	featureItem, ok := findSectionTypeConfig(configs, "feature_item")
	if !ok {
		t.Fatalf("feature_item config is missing")
	}

	if !containsString(featureItem.AllowedIn, "features") {
		t.Fatalf("feature_item should be allowed in features section, got: %v", featureItem.AllowedIn)
	}
	if containsString(featureItem.AllowedIn, "standard") {
		t.Fatalf("feature_item should not be allowed in standard section by default")
	}

	stepItem, ok := findSectionTypeConfig(configs, "step_item")
	if !ok {
		t.Fatalf("step_item config is missing")
	}

	if !containsString(stepItem.AllowedIn, "steps") {
		t.Fatalf("step_item should be allowed in steps section, got: %v", stepItem.AllowedIn)
	}
	if containsString(stepItem.AllowedIn, "standard") {
		t.Fatalf("step_item should not be allowed in standard section by default")
	}
}

func TestGetPageBuilderConfigWithOptionsRespectsFeatureFlags(t *testing.T) {
	service := &PageService{}

	config := service.GetPageBuilderConfigWithOptions(SectionCatalogOptions{
		BlogEnabled:    false,
		CoursesEnabled: false,
	})

	if hasSectionType(config.AvailableSections, "posts_list") {
		t.Fatalf("posts_list should be excluded from builder config when blog is disabled")
	}
	if hasSectionType(config.AvailableSections, "categories_list") {
		t.Fatalf("categories_list should be excluded from builder config when blog is disabled")
	}
	if hasSectionType(config.AvailableSections, "courses_list") {
		t.Fatalf("courses_list should be excluded from builder config when courses are disabled")
	}
	if hasSectionType(config.AvailableSections, "catalog") {
		t.Fatalf("catalog should be excluded from builder config when courses are disabled")
	}
}

func TestSectionCatalogIncludesTitlePositionSetting(t *testing.T) {
	catalog := NewSectionCatalog(nil, SectionCatalogOptions{
		BlogEnabled:    true,
		CoursesEnabled: true,
	})

	configs := catalog.SectionTypeConfigs()
	standard, ok := findSectionTypeConfig(configs, "standard")
	if !ok {
		t.Fatalf("standard section config is missing")
	}

	rawSetting, exists := standard.Schema["title_position"]
	if !exists {
		t.Fatalf("expected title_position setting to be present in schema")
	}

	setting, ok := rawSetting.(map[string]interface{})
	if !ok {
		t.Fatalf("title_position schema has invalid type: %T", rawSetting)
	}

	if fieldType, _ := setting["type"].(string); fieldType != "select" {
		t.Fatalf("expected title_position type to be select, got %q", fieldType)
	}

	options := parseSchemaOptionValues(setting["options"])
	if !containsString(options, "left") || !containsString(options, "center") || !containsString(options, "right") {
		t.Fatalf("title_position options are incomplete: %v", options)
	}

	rawDescriptionSetting, exists := standard.Schema["description_position"]
	if !exists {
		t.Fatalf("expected description_position setting to be present in schema")
	}

	descriptionSetting, ok := rawDescriptionSetting.(map[string]interface{})
	if !ok {
		t.Fatalf("description_position schema has invalid type: %T", rawDescriptionSetting)
	}

	if fieldType, _ := descriptionSetting["type"].(string); fieldType != "select" {
		t.Fatalf("expected description_position type to be select, got %q", fieldType)
	}

	descriptionOptions := parseSchemaOptionValues(descriptionSetting["options"])
	if !containsString(descriptionOptions, "left") || !containsString(descriptionOptions, "center") || !containsString(descriptionOptions, "right") {
		t.Fatalf("description_position options are incomplete: %v", descriptionOptions)
	}
}

func findSectionTypeConfig(configs []models.SectionTypeConfig, sectionType string) (models.SectionTypeConfig, bool) {
	for _, config := range configs {
		if config.Type == sectionType {
			return config, true
		}
	}
	return models.SectionTypeConfig{}, false
}

func hasSectionType(configs []models.SectionTypeConfig, sectionType string) bool {
	_, ok := findSectionTypeConfig(configs, sectionType)
	return ok
}

func parseSchemaOptionValues(raw interface{}) []string {
	switch typed := raw.(type) {
	case []map[string]string:
		values := make([]string, 0, len(typed))
		for _, option := range typed {
			if value, ok := option["value"]; ok && value != "" {
				values = append(values, value)
			}
		}
		return values
	case []interface{}:
		values := make([]string, 0, len(typed))
		for _, rawOption := range typed {
			option, ok := rawOption.(map[string]interface{})
			if !ok {
				continue
			}
			value, _ := option["value"].(string)
			if value != "" {
				values = append(values, value)
			}
		}
		return values
	default:
		return nil
	}
}

func parseSchemaStringValues(raw interface{}) []string {
	switch typed := raw.(type) {
	case []string:
		values := make([]string, 0, len(typed))
		for _, value := range typed {
			if value != "" {
				values = append(values, value)
			}
		}
		return values
	case []interface{}:
		values := make([]string, 0, len(typed))
		for _, rawValue := range typed {
			value, _ := rawValue.(string)
			if value != "" {
				values = append(values, value)
			}
		}
		return values
	default:
		return nil
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
