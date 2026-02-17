package sections

import (
	"html/template"
	"strings"
	"testing"

	"constructor-script-backend/internal/models"
)

type featuresTestRenderContext struct{}

func (featuresTestRenderContext) SanitizeHTML(input string) string {
	return input
}

func (featuresTestRenderContext) CloneTemplates() (*template.Template, error) {
	return template.New("features-test"), nil
}

func (featuresTestRenderContext) Services() ServiceProvider {
	return nil
}

func TestRenderFeaturesGlyphVariationUsesTitleWithoutText(t *testing.T) {
	ctx := featuresTestRenderContext{}
	elem := models.SectionElement{
		Content: models.Section{
			Variation: "glyph",
			Elements: []models.SectionElement{
				{
					Type: "feature_item",
					Content: map[string]interface{}{
						"title":     "Instant setup",
						"image_url": "https://example.com/icon.svg",
					},
				},
			},
		},
	}

	html, scripts := renderFeaturesSection(ctx, "page-view", elem)
	if len(scripts) != 0 {
		t.Fatalf("expected no scripts for features renderer, got %d", len(scripts))
	}
	if !strings.Contains(html, `class="page-view__feature-title">Instant setup</h3>`) {
		t.Fatalf("expected glyph variation to render title, got: %s", html)
	}
	if !strings.Contains(html, `alt="Instant setup"`) {
		t.Fatalf("expected glyph variation to use title as fallback alt text, got: %s", html)
	}
	if strings.Contains(html, `class="page-view__feature-text"`) {
		t.Fatalf("expected glyph variation to omit feature text markup, got: %s", html)
	}
}

func TestRenderFeaturesDefaultVariationStillRequiresText(t *testing.T) {
	ctx := featuresTestRenderContext{}
	elem := models.SectionElement{
		Content: models.Section{
			Variation: "cards",
			Elements: []models.SectionElement{
				{
					Type: "feature_item",
					Content: map[string]interface{}{
						"title":     "Instant setup",
						"image_url": "https://example.com/icon.svg",
					},
				},
			},
		},
	}

	html, _ := renderFeaturesSection(ctx, "page-view", elem)
	if html != "" {
		t.Fatalf("expected non glyph variation to skip items without text, got: %s", html)
	}
}

func TestRenderFeaturesGlyphVariationIgnoresTextContent(t *testing.T) {
	ctx := featuresTestRenderContext{}
	elem := models.SectionElement{
		Content: models.Section{
			Variation: "glyph",
			Elements: []models.SectionElement{
				{
					Type: "feature_item",
					Content: map[string]interface{}{
						"title":     "Unified search",
						"text":      "Should stay hidden for glyph variation",
						"image_url": "https://example.com/icon.svg",
					},
				},
			},
		},
	}

	html, _ := renderFeaturesSection(ctx, "page-view", elem)
	if strings.Contains(html, "Should stay hidden for glyph variation") {
		t.Fatalf("expected glyph variation to suppress text output, got: %s", html)
	}
	if strings.Contains(html, `class="page-view__feature-text"`) {
		t.Fatalf("expected glyph variation to omit feature text node, got: %s", html)
	}
}

func TestRenderFeaturesConstellationVariationRendersSubtitleAndText(t *testing.T) {
	ctx := featuresTestRenderContext{}
	elem := models.SectionElement{
		Content: models.Section{
			Variation: "constellation",
			Elements: []models.SectionElement{
				{
					Type: "feature_item",
					Content: map[string]interface{}{
						"title":     "Unified search",
						"subtitle":  "Across docs, courses, and posts",
						"text":      "Find the exact answer with one query.",
						"image_url": "https://example.com/icon.svg",
					},
				},
			},
		},
	}

	html, _ := renderFeaturesSection(ctx, "page-view", elem)
	if !strings.Contains(html, `class="page-view__feature-head"`) {
		t.Fatalf("expected constellation variation to render feature head row, got: %s", html)
	}
	if !strings.Contains(html, `class="page-view__feature-meta"`) {
		t.Fatalf("expected constellation variation to render feature meta block, got: %s", html)
	}
	if !strings.Contains(html, `class="page-view__feature-title">Unified search</h3>`) {
		t.Fatalf("expected constellation variation to render title, got: %s", html)
	}
	if !strings.Contains(html, `class="page-view__feature-subtitle">Across docs, courses, and posts</p>`) {
		t.Fatalf("expected constellation variation to render subtitle, got: %s", html)
	}
	if !strings.Contains(html, `class="page-view__feature-text">Find the exact answer with one query.</p>`) {
		t.Fatalf("expected constellation variation to render text, got: %s", html)
	}
	headIndex := strings.Index(html, `class="page-view__feature-head"`)
	textIndex := strings.Index(html, `class="page-view__feature-text"`)
	if headIndex == -1 || textIndex == -1 || headIndex > textIndex {
		t.Fatalf("expected constellation variation to render text after head row, got: %s", html)
	}
}

func TestRenderFeaturesConstellationVariationRequiresSubtitle(t *testing.T) {
	ctx := featuresTestRenderContext{}
	elem := models.SectionElement{
		Content: models.Section{
			Variation: "constellation",
			Elements: []models.SectionElement{
				{
					Type: "feature_item",
					Content: map[string]interface{}{
						"title":     "Unified search",
						"text":      "Find the exact answer with one query.",
						"image_url": "https://example.com/icon.svg",
					},
				},
			},
		},
	}

	html, _ := renderFeaturesSection(ctx, "page-view", elem)
	if html != "" {
		t.Fatalf("expected constellation variation to skip items without subtitle, got: %s", html)
	}
}
