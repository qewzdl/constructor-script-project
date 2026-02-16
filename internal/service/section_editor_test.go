package service

import (
	"errors"
	"testing"

	"constructor-script-backend/internal/models"
)

func TestSectionEditorReorderValidatesPayload(t *testing.T) {
	editor := NewSectionEditor([]models.Section{
		{ID: "first", Type: "standard"},
		{ID: "second", Type: "standard"},
	}, nil)

	if err := editor.Reorder([]string{"second"}); err == nil {
		t.Fatalf("expected error when reorder payload length does not match sections length")
	}

	if err := editor.Reorder([]string{"second", "missing"}); err == nil {
		t.Fatalf("expected error for unknown section id")
	}

	if err := editor.Reorder([]string{"second", "second"}); err == nil {
		t.Fatalf("expected error for duplicated section id in payload")
	}

	if err := editor.Reorder([]string{"second", "first"}); err != nil {
		t.Fatalf("expected reorder to succeed, got error: %v", err)
	}

	prepared, err := editor.Build(PrepareSectionsOptions{NormaliseSpacing: true})
	if err != nil {
		t.Fatalf("expected build to succeed after reorder: %v", err)
	}
	if len(prepared) != 2 {
		t.Fatalf("expected 2 sections after reorder, got %d", len(prepared))
	}
	if prepared[0].ID != "second" || prepared[1].ID != "first" {
		t.Fatalf("unexpected reorder result: %#v", []string{prepared[0].ID, prepared[1].ID})
	}
}

func TestSectionEditorDuplicateDeepCopiesElementContent(t *testing.T) {
	editor := NewSectionEditor([]models.Section{
		{
			ID:   "source",
			Type: "standard",
			Elements: []models.SectionElement{
				{
					ID:      "elem-1",
					Type:    "paragraph",
					Content: map[string]interface{}{"text": "Initial"},
				},
			},
		},
	}, nil)

	if err := editor.Duplicate("source"); err != nil {
		t.Fatalf("expected duplicate to succeed, got: %v", err)
	}

	sections := editor.Sections()
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections after duplication, got %d", len(sections))
	}
	if sections[0].ID == sections[1].ID {
		t.Fatalf("expected duplicated section to have a different id")
	}
	if sections[0].Elements[0].ID == sections[1].Elements[0].ID {
		t.Fatalf("expected duplicated element to have a different id")
	}

	originalContent, ok := sections[0].Elements[0].Content.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected original content type: %T", sections[0].Elements[0].Content)
	}
	duplicateContent, ok := sections[1].Elements[0].Content.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected duplicate content type: %T", sections[1].Elements[0].Content)
	}

	duplicateContent["text"] = "Changed"
	if originalContent["text"] == "Changed" {
		t.Fatalf("expected duplicate content mutation not to affect original content")
	}
}

func TestSectionEditorSupportsDynamicSettingsLifecycle(t *testing.T) {
	editor := NewSectionEditor(nil, nil)

	if err := editor.Add(models.AddSectionRequest{
		Type:      "hero",
		Title:     "Hero",
		Settings:  map[string]interface{}{"title": "Welcome"},
		Image:     "  /uploads/hero.jpg ",
		Variation: "minimal",
	}); err != nil {
		t.Fatalf("expected add to succeed, got: %v", err)
	}

	sections := editor.Sections()
	if len(sections) != 1 {
		t.Fatalf("expected 1 section after add, got %d", len(sections))
	}
	sectionID := sections[0].ID

	updatedSettings := map[string]interface{}{
		"title":       "Updated",
		"button_text": "Open",
	}
	image := "/uploads/hero-updated.jpg"
	variation := "centered"

	if err := editor.Update(sectionID, models.UpdateSectionRequest{
		Settings:  &updatedSettings,
		Image:     &image,
		Variation: &variation,
	}); err != nil {
		t.Fatalf("expected update to succeed, got: %v", err)
	}

	prepared, err := editor.Build(PrepareSectionsOptions{NormaliseSpacing: true})
	if err != nil {
		t.Fatalf("expected build to succeed, got: %v", err)
	}
	if len(prepared) != 1 {
		t.Fatalf("expected 1 section after update, got %d", len(prepared))
	}
	if prepared[0].Settings["title"] != "Updated" {
		t.Fatalf("expected updated section setting to be preserved, got %#v", prepared[0].Settings["title"])
	}
	if prepared[0].Image != "/uploads/hero-updated.jpg" {
		t.Fatalf("expected image to be updated and trimmed, got %q", prepared[0].Image)
	}
	if prepared[0].Variation != "centered" {
		t.Fatalf("expected variation to be normalised by theme definitions, got %q", prepared[0].Variation)
	}
}

func TestSectionEditorReturnsSectionNotFoundError(t *testing.T) {
	editor := NewSectionEditor([]models.Section{
		{ID: "first", Type: "standard"},
	}, nil)

	if err := editor.Delete("missing"); !errors.Is(err, errSectionNotFound) {
		t.Fatalf("expected errSectionNotFound for missing section delete, got %v", err)
	}
	if err := editor.Update("missing", models.UpdateSectionRequest{}); !errors.Is(err, errSectionNotFound) {
		t.Fatalf("expected errSectionNotFound for missing section update, got %v", err)
	}
	if err := editor.Duplicate("missing"); !errors.Is(err, errSectionNotFound) {
		t.Fatalf("expected errSectionNotFound for missing section duplicate, got %v", err)
	}
}
