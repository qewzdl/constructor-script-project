package sections

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"

	"constructor-script-backend/internal/models"
)

// RegisterSteps registers the steps section and step item renderers.
func RegisterSteps(reg *Registry) {
	if reg == nil {
		return
	}

	reg.RegisterSafe("steps", renderStepsSection)
	reg.RegisterSafe("step_item", renderStepItem)
}

// RegisterStepsWithMetadata registers the steps section with metadata support.
func RegisterStepsWithMetadata(reg *RegistryWithMetadata) {
	if reg == nil {
		return
	}

	reg.Registry.RegisterSafe("step_item", renderStepItem)

	desc := &SectionDescriptor{
		Renderer: renderStepsSection,
		Metadata: SectionMetadata{
			Type:        "steps",
			Name:        "Steps",
			Description: "Present an ordered step-by-step process with clear titles and details.",
			Category:    "marketing",
			Icon:        "list",
		},
	}

	reg.RegisterWithMetadata(desc)
}

func renderStepsSection(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	section, ok := extractSection(elem)
	if !ok {
		return "", nil
	}

	if len(section.Elements) == 0 {
		return "", nil
	}

	containerClass := fmt.Sprintf("%s__steps", prefix)
	listClass := fmt.Sprintf("%s__steps-list", prefix)

	var items []string
	for _, item := range section.Elements {
		if strings.TrimSpace(strings.ToLower(item.Type)) != "step_item" {
			continue
		}
		stepPosition := len(items) + 1
		itemHTML, _ := renderStepItemWithPosition(ctx, prefix, item, stepPosition)
		if itemHTML != "" {
			items = append(items, itemHTML)
		}
	}

	if len(items) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString(`<div class="` + containerClass + `">`)
	sb.WriteString(`<div class="` + listClass + `">`)
	for _, item := range items {
		sb.WriteString(item)
	}
	sb.WriteString(`</div>`)
	sb.WriteString(`</div>`)

	return sb.String(), nil
}

func renderStepItem(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	return renderStepItemWithPosition(ctx, prefix, elem, 0)
}

func renderStepItemWithPosition(ctx RenderContext, prefix string, elem models.SectionElement, position int) (string, []string) {
	content := sectionContent(elem)

	number := strings.TrimSpace(getString(content, "number"))
	title := strings.TrimSpace(getString(content, "title"))
	text := strings.TrimSpace(getString(content, "text"))

	if title == "" || text == "" {
		return "", nil
	}

	if number == "" {
		if position <= 0 && elem.Order > 0 {
			position = elem.Order
		}
		if position > 0 {
			number = strconv.Itoa(position)
		}
	}

	itemClass := fmt.Sprintf("%s__step-item", prefix)
	numberClass := fmt.Sprintf("%s__step-number", prefix)
	bodyClass := fmt.Sprintf("%s__step-body", prefix)
	titleClass := fmt.Sprintf("%s__step-title", prefix)
	textClass := fmt.Sprintf("%s__step-text", prefix)

	var sb strings.Builder
	sb.WriteString(`<article class="` + itemClass + `">`)
	if number != "" {
		sb.WriteString(`<span class="` + numberClass + `">` + template.HTMLEscapeString(number) + `</span>`)
	}
	sb.WriteString(`<div class="` + bodyClass + `">`)
	sb.WriteString(`<h3 class="` + titleClass + `">` + template.HTMLEscapeString(title) + `</h3>`)
	sb.WriteString(`<p class="` + textClass + `">` + ctx.SanitizeHTML(text) + `</p>`)
	sb.WriteString(`</div>`)
	sb.WriteString(`</article>`)

	return sb.String(), nil
}
