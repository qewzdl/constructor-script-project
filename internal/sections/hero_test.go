package sections

import (
	"html/template"
	"strings"
	"testing"

	"constructor-script-backend/internal/models"
)

type heroTestRenderContext struct{}

func (heroTestRenderContext) SanitizeHTML(input string) string {
	return input
}

func (heroTestRenderContext) CloneTemplates() (*template.Template, error) {
	return template.New("hero-test"), nil
}

func (heroTestRenderContext) Services() ServiceProvider {
	return nil
}

func TestRenderHeroCenteredVariationUsesDedicatedStructure(t *testing.T) {
	ctx := heroTestRenderContext{}
	elem := models.SectionElement{
		Content: models.Section{
			Variation: "centered",
			Settings: map[string]interface{}{
				"title":       "Welcome",
				"subtitle":    "Start here",
				"text":        "Intro text",
				"image_url":   "https://example.com/hero.jpg",
				"button_text": "Get started",
				"button_url":  "/start",
			},
		},
	}

	html, scripts := renderHero(ctx, "page-view", elem)
	if len(scripts) != 0 {
		t.Fatalf("expected hero renderer to return no scripts, got %d", len(scripts))
	}
	if !strings.Contains(
		html,
		`class="page-view__hero-container page-view__hero-container--centered"`,
	) {
		t.Fatalf("expected centered hero container class, got: %s", html)
	}
	if !strings.Contains(
		html,
		`<figure class="page-view__hero-image page-view__hero-image--centered">`,
	) {
		t.Fatalf("expected centered hero image figure wrapper, got: %s", html)
	}
}

func TestRenderHeroMinimalVariationOmitsImageMarkup(t *testing.T) {
	ctx := heroTestRenderContext{}
	elem := models.SectionElement{
		Content: models.Section{
			Variation: "minimal",
			Settings: map[string]interface{}{
				"title":       "Welcome",
				"image_url":   "https://example.com/hero.jpg",
				"button_text": "Get started",
				"button_url":  "/start",
			},
		},
	}

	html, _ := renderHero(ctx, "page-view", elem)
	if strings.Contains(html, "page-view__hero-image-img") {
		t.Fatalf("expected minimal variation to omit image markup, got: %s", html)
	}
}

func TestRenderHeroImmersiveVariationUsesLayeredStructure(t *testing.T) {
	ctx := heroTestRenderContext{}
	elem := models.SectionElement{
		Content: models.Section{
			Variation: "immersive",
			Settings: map[string]interface{}{
				"title":                 "Welcome",
				"subtitle":              "Next-level launch",
				"text":                  "Build pages with a bold visual first impression.",
				"image_url":             "https://example.com/hero.jpg",
				"button_text":           "Explore",
				"button_url":            "/explore",
				"button_icon":           "⚡",
				"secondary_button_text": "See pricing",
				"secondary_button_url":  "/pricing",
				"secondary_button_icon": "↗",
			},
		},
	}

	html, scripts := renderHero(ctx, "page-view", elem)
	if len(scripts) != 1 || scripts[0] != "/static/js/hero-immersive-cursor.js" {
		t.Fatalf("expected immersive variation to return cursor script, got: %v", scripts)
	}
	if !strings.Contains(html, `class="page-view__hero-stage"`) {
		t.Fatalf("expected immersive variation to include stage wrapper, got: %s", html)
	}
	if strings.Contains(html, `class="page-view__hero-media"`) {
		t.Fatalf("expected immersive variation to omit media layer, got: %s", html)
	}
	if strings.Contains(html, "page-view__hero-image-img") {
		t.Fatalf("expected immersive variation to omit image markup, got: %s", html)
	}
	if !strings.Contains(html, `class="page-view__hero-container page-view__hero-container--immersive"`) {
		t.Fatalf("expected immersive variation to include immersive container class, got: %s", html)
	}
	if !strings.Contains(html, `class="page-view__hero-content page-view__hero-content--immersive"`) {
		t.Fatalf("expected immersive variation to include immersive content class, got: %s", html)
	}
	if !strings.Contains(html, `class="page-view__hero-actions"`) {
		t.Fatalf("expected immersive variation to include actions wrapper, got: %s", html)
	}
	if !strings.Contains(html, `class="page-view__hero-button page-view__hero-button--primary"`) {
		t.Fatalf("expected immersive variation to include primary button class, got: %s", html)
	}
	if !strings.Contains(html, `class="page-view__hero-button page-view__hero-button--secondary"`) {
		t.Fatalf("expected immersive variation to include secondary button class, got: %s", html)
	}
	if !strings.Contains(html, `<span class="page-view__hero-button-icon" aria-hidden="true">⚡</span>`) {
		t.Fatalf("expected immersive variation to include primary button icon, got: %s", html)
	}
	if !strings.Contains(html, `<span class="page-view__hero-button-icon" aria-hidden="true">↗</span>`) {
		t.Fatalf("expected immersive variation to include secondary button icon, got: %s", html)
	}
	subtitleIndex := strings.Index(html, `class="page-view__hero-subtitle"`)
	titleIndex := strings.Index(html, `class="page-view__hero-title"`)
	textIndex := strings.Index(html, `class="page-view__hero-text"`)
	if subtitleIndex == -1 || titleIndex == -1 || textIndex == -1 || subtitleIndex > titleIndex || titleIndex > textIndex {
		t.Fatalf("expected immersive content order subtitle -> title -> text, got: %s", html)
	}
}

func TestRenderHeroImmersiveVariationEscapesButtonIcons(t *testing.T) {
	ctx := heroTestRenderContext{}
	elem := models.SectionElement{
		Content: models.Section{
			Variation: "immersive",
			Settings: map[string]interface{}{
				"title":                 "Welcome",
				"button_text":           "Explore",
				"button_url":            "/explore",
				"button_icon":           "<svg>",
				"secondary_button_text": "See pricing",
				"secondary_button_url":  "/pricing",
				"secondary_button_icon": "<img src=x onerror=alert(1)>",
			},
		},
	}

	html, _ := renderHero(ctx, "page-view", elem)
	if !strings.Contains(html, `&lt;svg&gt;`) {
		t.Fatalf("expected primary icon to be escaped, got: %s", html)
	}
	if !strings.Contains(html, `&lt;img src=x onerror=alert(1)&gt;`) {
		t.Fatalf("expected secondary icon to be escaped, got: %s", html)
	}
}
