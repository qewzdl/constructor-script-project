package service

import (
	"strings"
	"testing"

	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
)

func TestPrepareSections_AllowsConfiguredElements(t *testing.T) {
	sections := []models.Section{
		{
			Type: "features",
			Elements: []models.SectionElement{
				{Type: "feature_item", Content: map[string]interface{}{"text": "Feature"}},
			},
		},
	}

	prepared, err := PrepareSections(sections, nil, PrepareSectionsOptions{})
	if err != nil {
		t.Fatalf("expected sections to be prepared, got error: %v", err)
	}
	if len(prepared) != 1 {
		t.Fatalf("expected 1 section, got %d", len(prepared))
	}
	if len(prepared[0].Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(prepared[0].Elements))
	}
	if prepared[0].Elements[0].Type != "feature_item" {
		t.Fatalf("expected element type to remain normalised, got %q", prepared[0].Elements[0].Type)
	}
}

func TestPrepareSections_RejectsDisallowedElements(t *testing.T) {
	sections := []models.Section{
		{
			Type: "features",
			Elements: []models.SectionElement{
				{Type: "image", Content: map[string]interface{}{"url": "https://example.com/img.png"}},
			},
		},
	}

	_, err := PrepareSections(sections, nil, PrepareSectionsOptions{})
	if err == nil {
		t.Fatalf("expected error for disallowed element type, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "not allowed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestPrepareSections_NormalisesSectionVariation(t *testing.T) {
	sections := []models.Section{
		{
			Type:      "features",
			Variation: "unknown-variation",
			Elements: []models.SectionElement{
				{Type: "feature_item", Content: map[string]interface{}{"text": "Feature"}},
			},
		},
	}

	prepared, err := PrepareSections(sections, nil, PrepareSectionsOptions{})
	if err != nil {
		t.Fatalf("expected sections to be prepared, got error: %v", err)
	}
	if len(prepared) != 1 {
		t.Fatalf("expected 1 section, got %d", len(prepared))
	}
	if prepared[0].Variation != "cards" {
		t.Fatalf("expected default features variation 'cards', got %q", prepared[0].Variation)
	}
}

func TestPrepareSections_NormalisesStepsVariation(t *testing.T) {
	sections := []models.Section{
		{
			Type:      "steps",
			Variation: "custom-variation",
			Elements: []models.SectionElement{
				{
					Type: "step_item",
					Content: map[string]interface{}{
						"title": "Step title",
						"text":  "Step text",
					},
				},
			},
		},
	}

	prepared, err := PrepareSections(sections, nil, PrepareSectionsOptions{})
	if err != nil {
		t.Fatalf("expected sections to be prepared, got error: %v", err)
	}
	if len(prepared) != 1 {
		t.Fatalf("expected 1 section, got %d", len(prepared))
	}
	if prepared[0].Variation != "numbered" {
		t.Fatalf("expected default steps variation 'numbered', got %q", prepared[0].Variation)
	}
}

func TestPrepareSections_UsesVariationFromSettingsFallback(t *testing.T) {
	sections := []models.Section{
		{
			Type: "hero",
			Settings: map[string]interface{}{
				"variation": "minimal",
			},
		},
	}

	prepared, err := PrepareSections(sections, nil, PrepareSectionsOptions{})
	if err != nil {
		t.Fatalf("expected sections to be prepared, got error: %v", err)
	}
	if len(prepared) != 1 {
		t.Fatalf("expected 1 section, got %d", len(prepared))
	}
	if prepared[0].Variation != "minimal" {
		t.Fatalf("expected hero variation 'minimal', got %q", prepared[0].Variation)
	}
}

func TestPrepareSections_NormalisesSideSpacingValues(t *testing.T) {
	paddingTop := 33
	paddingBottom := 7
	marginTop := -5
	marginBottom := 129
	sections := []models.Section{
		{
			Type:          "standard",
			PaddingTop:    &paddingTop,
			PaddingBottom: &paddingBottom,
			MarginTop:     &marginTop,
			MarginBottom:  &marginBottom,
		},
	}

	prepared, err := PrepareSections(sections, nil, PrepareSectionsOptions{
		NormaliseSpacing: true,
	})
	if err != nil {
		t.Fatalf("expected sections to be prepared, got error: %v", err)
	}
	if len(prepared) != 1 {
		t.Fatalf("expected 1 section, got %d", len(prepared))
	}

	section := prepared[0]
	if section.PaddingTop == nil || *section.PaddingTop != 32 {
		t.Fatalf("expected padding top to normalise to 32, got %#v", section.PaddingTop)
	}
	if section.PaddingBottom == nil || *section.PaddingBottom != 8 {
		t.Fatalf("expected padding bottom to normalise to 8, got %#v", section.PaddingBottom)
	}
	if section.MarginTop == nil || *section.MarginTop != 0 {
		t.Fatalf("expected margin top to normalise to 0, got %#v", section.MarginTop)
	}
	if section.MarginBottom == nil || *section.MarginBottom != 128 {
		t.Fatalf("expected margin bottom to normalise to 128, got %#v", section.MarginBottom)
	}
}

func TestPrepareSections_NormalisesSectionSettingsUsingDefinitions(t *testing.T) {
	sections := []models.Section{
		{
			Type: "catalog",
			Settings: map[string]interface{}{
				"Display Mode":      "PAGINATED",
				"show_all_button":   "yes",
				"carousel_columns":  "12",
				"all_courses_url":   "  /courses/all  ",
				"all_courses_label": "  Browse all courses  ",
				"custom_hint":       "keep-me",
			},
		},
	}

	prepared, err := PrepareSections(sections, nil, PrepareSectionsOptions{})
	if err != nil {
		t.Fatalf("expected sections to be prepared, got error: %v", err)
	}
	if len(prepared) != 1 {
		t.Fatalf("expected 1 section, got %d", len(prepared))
	}

	section := prepared[0]
	if section.Limit != constants.DefaultCourseListSectionLimit {
		t.Fatalf("expected default course list limit %d, got %d", constants.DefaultCourseListSectionLimit, section.Limit)
	}
	if section.Settings == nil {
		t.Fatalf("expected normalised settings to be present")
	}

	displayMode, _ := section.Settings["display_mode"].(string)
	if displayMode != constants.CourseListDisplayPaginated {
		t.Fatalf("expected display_mode=%q, got %q", constants.CourseListDisplayPaginated, displayMode)
	}

	showAll, ok := section.Settings["show_all_button"].(bool)
	if !ok {
		t.Fatalf("expected show_all_button bool value, got %T", section.Settings["show_all_button"])
	}
	if !showAll {
		t.Fatalf("expected show_all_button to be true")
	}

	carouselColumns, ok := section.Settings["carousel_columns"].(int)
	if !ok {
		t.Fatalf("expected carousel_columns int value, got %T", section.Settings["carousel_columns"])
	}
	if carouselColumns != constants.MaxCarouselColumns {
		t.Fatalf("expected carousel_columns to be clamped to %d, got %d", constants.MaxCarouselColumns, carouselColumns)
	}

	allCoursesURL, _ := section.Settings["all_courses_url"].(string)
	if allCoursesURL != "/courses/all" {
		t.Fatalf("expected all_courses_url to be trimmed, got %q", allCoursesURL)
	}

	if customHint, _ := section.Settings["custom_hint"].(string); customHint != "keep-me" {
		t.Fatalf("expected unknown setting to be preserved, got %q", customHint)
	}
}

func TestPrepareSections_NormalisesTitlePositionSetting(t *testing.T) {
	sections := []models.Section{
		{
			Type: "standard",
			Settings: map[string]interface{}{
				"title_position":       "CENTER",
				"description_position": "RIGHT",
			},
		},
	}

	prepared, err := PrepareSections(sections, nil, PrepareSectionsOptions{})
	if err != nil {
		t.Fatalf("expected sections to be prepared, got error: %v", err)
	}
	if len(prepared) != 1 {
		t.Fatalf("expected 1 section, got %d", len(prepared))
	}
	if prepared[0].Settings == nil {
		t.Fatalf("expected normalised settings to be present")
	}
	position, _ := prepared[0].Settings["title_position"].(string)
	if position != "center" {
		t.Fatalf("expected title_position to normalise to %q, got %q", "center", position)
	}
	descriptionPosition, _ := prepared[0].Settings["description_position"].(string)
	if descriptionPosition != "right" {
		t.Fatalf("expected description_position to normalise to %q, got %q", "right", descriptionPosition)
	}
}

func TestPrepareSections_DefaultsTitlePositionSetting(t *testing.T) {
	sections := []models.Section{
		{
			Type: "standard",
		},
	}

	prepared, err := PrepareSections(sections, nil, PrepareSectionsOptions{})
	if err != nil {
		t.Fatalf("expected sections to be prepared, got error: %v", err)
	}
	if len(prepared) != 1 {
		t.Fatalf("expected 1 section, got %d", len(prepared))
	}
	if prepared[0].Settings == nil {
		t.Fatalf("expected settings to include defaults")
	}
	position, _ := prepared[0].Settings["title_position"].(string)
	if position != "left" {
		t.Fatalf("expected default title_position to be %q, got %q", "left", position)
	}
	descriptionPosition, _ := prepared[0].Settings["description_position"].(string)
	if descriptionPosition != "left" {
		t.Fatalf("expected default description_position to be %q, got %q", "left", descriptionPosition)
	}
}
