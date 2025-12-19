package service

import (
	"strings"
	"testing"

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
