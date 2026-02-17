package sections

import (
	"html/template"
	"strings"
	"testing"

	"constructor-script-backend/internal/models"
)

type stepsTestRenderContext struct{}

func (stepsTestRenderContext) SanitizeHTML(input string) string {
	return input
}

func (stepsTestRenderContext) CloneTemplates() (*template.Template, error) {
	return template.New("steps-test"), nil
}

func (stepsTestRenderContext) Services() ServiceProvider {
	return nil
}

func TestRenderStepsNumberedVariationRendersNumberTitleAndText(t *testing.T) {
	ctx := stepsTestRenderContext{}
	elem := models.SectionElement{
		Content: models.Section{
			Variation: "numbered",
			Elements: []models.SectionElement{
				{
					Type: "step_item",
					Content: map[string]interface{}{
						"number": "01",
						"title":  "Plan the rollout",
						"text":   "Define milestones and assign owners before launch.",
					},
				},
			},
		},
	}

	html, scripts := renderStepsSection(ctx, "page-view", elem)
	if len(scripts) != 0 {
		t.Fatalf("expected no scripts for steps renderer, got %d", len(scripts))
	}
	if !strings.Contains(html, `class="page-view__step-number">01</span>`) {
		t.Fatalf("expected steps variation to render step number, got: %s", html)
	}
	if !strings.Contains(html, `class="page-view__step-title">Plan the rollout</h3>`) {
		t.Fatalf("expected steps variation to render step title, got: %s", html)
	}
	if !strings.Contains(html, `class="page-view__step-text">Define milestones and assign owners before launch.</p>`) {
		t.Fatalf("expected steps variation to render step text, got: %s", html)
	}
}

func TestRenderStepsFallsBackToSequentialNumber(t *testing.T) {
	ctx := stepsTestRenderContext{}
	elem := models.SectionElement{
		Content: models.Section{
			Variation: "numbered",
			Elements: []models.SectionElement{
				{
					Type: "step_item",
					Content: map[string]interface{}{
						"title": "Gather requirements",
						"text":  "Collect goals from product and support teams.",
					},
				},
				{
					Type: "step_item",
					Content: map[string]interface{}{
						"title": "Launch and monitor",
						"text":  "Ship to production and watch metrics.",
					},
				},
			},
		},
	}

	html, _ := renderStepsSection(ctx, "page-view", elem)
	if !strings.Contains(html, `class="page-view__step-number">1</span>`) {
		t.Fatalf("expected first step to use sequential fallback number, got: %s", html)
	}
	if !strings.Contains(html, `class="page-view__step-number">2</span>`) {
		t.Fatalf("expected second step to use sequential fallback number, got: %s", html)
	}
}

func TestRenderStepsRequiresTitleAndText(t *testing.T) {
	ctx := stepsTestRenderContext{}
	elem := models.SectionElement{
		Content: models.Section{
			Variation: "numbered",
			Elements: []models.SectionElement{
				{
					Type: "step_item",
					Content: map[string]interface{}{
						"title": "Incomplete step",
					},
				},
			},
		},
	}

	html, _ := renderStepsSection(ctx, "page-view", elem)
	if html != "" {
		t.Fatalf("expected steps variation to skip items without text, got: %s", html)
	}
}
