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
