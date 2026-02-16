package sections

import (
	"testing"

	"constructor-script-backend/internal/models"
)

func TestRegistryWithMetadata_NormalisesAndSortsMetadata(t *testing.T) {
	registry := NewRegistryWithMetadata()

	if err := registry.RegisterWithMetadata(&SectionDescriptor{
		Renderer: testSectionRenderer,
		Metadata: SectionMetadata{
			Type:            " Hero ",
			Order:           2,
			AllowedIn:       []string{"Page", "page", "Homepage"},
			AllowedElements: []string{"Paragraph", "paragraph"},
		},
	}); err != nil {
		t.Fatalf("expected hero descriptor to register, got error: %v", err)
	}

	if err := registry.RegisterWithMetadata(&SectionDescriptor{
		Renderer: testSectionRenderer,
		Metadata: SectionMetadata{
			Type:  "contact",
			Order: 1,
		},
	}); err != nil {
		t.Fatalf("expected contact descriptor to register, got error: %v", err)
	}

	list := registry.ListMetadata()
	if len(list) != 2 {
		t.Fatalf("expected 2 metadata records, got %d", len(list))
	}
	if list[0].Type != "contact" || list[1].Type != "hero" {
		t.Fatalf("expected metadata to be sorted by order then type, got %#v", []string{list[0].Type, list[1].Type})
	}

	hero := list[1]
	if len(hero.AllowedIn) != 2 {
		t.Fatalf("expected allowed_in values to be deduplicated, got %#v", hero.AllowedIn)
	}
	if len(hero.AllowedElements) != 1 || hero.AllowedElements[0] != "paragraph" {
		t.Fatalf("expected allowed_elements to be normalised, got %#v", hero.AllowedElements)
	}
}

func TestSectionBuilder_BuildsExtendedMetadata(t *testing.T) {
	desc, err := NewSectionBuilder("Plugin_Cards").
		WithRenderer(testSectionRenderer).
		WithOrder(7).
		WithAllowedIn("Page", "homepage", "PAGE").
		WithAllowedElements("Paragraph", "image", "paragraph").
		WithSupportsElements(true).
		WithSupportsHeaderImage(true).
		Build()
	if err != nil {
		t.Fatalf("expected descriptor to build, got error: %v", err)
	}

	if desc.Metadata.Type != "plugin_cards" {
		t.Fatalf("expected normalised type plugin_cards, got %q", desc.Metadata.Type)
	}
	if desc.Metadata.Order != 7 {
		t.Fatalf("expected order 7, got %d", desc.Metadata.Order)
	}
	if len(desc.Metadata.AllowedIn) != 2 {
		t.Fatalf("expected allowed_in to be normalised, got %#v", desc.Metadata.AllowedIn)
	}
	if len(desc.Metadata.AllowedElements) != 2 {
		t.Fatalf("expected allowed_elements to be normalised, got %#v", desc.Metadata.AllowedElements)
	}
	if desc.Metadata.SupportsElements == nil || !*desc.Metadata.SupportsElements {
		t.Fatalf("expected supports_elements=true")
	}
	if desc.Metadata.SupportsHeaderImage == nil || !*desc.Metadata.SupportsHeaderImage {
		t.Fatalf("expected supports_header_image=true")
	}
}

func testSectionRenderer(RenderContext, string, models.SectionElement) (string, []string) {
	return "", nil
}
