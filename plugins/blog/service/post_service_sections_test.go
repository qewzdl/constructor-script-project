package blogservice

import (
	"strings"
	"testing"

	"constructor-script-backend/internal/models"
)

func TestPostServicePrepareSectionsRejectsDisallowedElements(t *testing.T) {
	svc := &PostService{}
	sections := []models.Section{
		{
			Type: "features",
			Elements: []models.SectionElement{
				{Type: "image", Content: map[string]interface{}{"url": "https://example.com/image.png"}},
			},
		},
	}

	_, err := svc.prepareSections(sections)
	if err == nil {
		t.Fatalf("expected error for disallowed element type, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "not allowed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestPostServicePrepareSectionsAllowsConfiguredElements(t *testing.T) {
	svc := &PostService{}
	sections := []models.Section{
		{
			Type: "features",
			Elements: []models.SectionElement{
				{Type: "feature_item", Content: map[string]interface{}{"text": "Feature"}},
			},
		},
	}

	processed, err := svc.prepareSections(sections)
	if err != nil {
		t.Fatalf("expected sections to be prepared, got error: %v", err)
	}
	if len(processed) != 1 {
		t.Fatalf("expected 1 section, got %d", len(processed))
	}
	if len(processed[0].Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(processed[0].Elements))
	}
	if processed[0].Elements[0].Type != "feature_item" {
		t.Fatalf("unexpected element type: %s", processed[0].Elements[0].Type)
	}
}
