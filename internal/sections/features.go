package sections

import (
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
)

// RegisterFeatures registers the features section and feature item renderers.
func RegisterFeatures(reg *Registry) {
	if reg == nil {
		return
	}

	reg.RegisterSafe("features", renderFeaturesSection)
	reg.RegisterSafe("feature_item", renderFeatureItem)
}

// RegisterFeaturesWithMetadata registers the features section with metadata support.
func RegisterFeaturesWithMetadata(reg *RegistryWithMetadata) {
	if reg == nil {
		return
	}

	// Feature items are elements, so we register them directly on the underlying registry.
	reg.Registry.RegisterSafe("feature_item", renderFeatureItem)

	desc := &SectionDescriptor{
		Renderer: renderFeaturesSection,
		Metadata: SectionMetadata{
			Type:        "features",
			Name:        "Features",
			Description: "Showcase key features with supporting images.",
			Category:    "marketing",
			Icon:        "sparkles",
		},
	}

	reg.RegisterWithMetadata(desc)
}

func renderFeaturesSection(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	section, ok := extractSection(elem)
	if !ok {
		return "", nil
	}

	if len(section.Elements) == 0 {
		return "", nil
	}

	containerClass := fmt.Sprintf("%s__features", prefix)
	listClass := fmt.Sprintf("%s__features-list", prefix)

	var items []string
	for _, item := range section.Elements {
		if strings.TrimSpace(strings.ToLower(item.Type)) != "feature_item" {
			continue
		}
		itemHTML, _ := renderFeatureItem(ctx, prefix, item)
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

func renderFeatureItem(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)

	title := strings.TrimSpace(getString(content, "title"))
	text := strings.TrimSpace(getString(content, "text"))
	imageURL := strings.TrimSpace(getString(content, "image_url"))
	imageAlt := strings.TrimSpace(getString(content, "image_alt"))

	if text == "" {
		return "", nil
	}

	itemClass := fmt.Sprintf("%s__feature-item", prefix)
	bodyClass := fmt.Sprintf("%s__feature-body", prefix)
	mediaClass := fmt.Sprintf("%s__feature-media", prefix)
	imageClass := fmt.Sprintf("%s__feature-image", prefix)
	titleClass := fmt.Sprintf("%s__feature-title", prefix)
	textClass := fmt.Sprintf("%s__feature-text", prefix)

	var sb strings.Builder
	sb.WriteString(`<article class="` + itemClass + `">`)

	if imageURL != "" {
		alt := imageAlt
		if alt == "" {
			alt = text
		}
		sb.WriteString(`<div class="` + mediaClass + `">`)
		sb.WriteString(`<img class="` + imageClass + `" src="` + template.HTMLEscapeString(imageURL) + `" alt="` + template.HTMLEscapeString(alt) + `" />`)
		sb.WriteString(`</div>`)
	}

	if text != "" {
		sb.WriteString(`<div class="` + bodyClass + `">`)
		if title != "" {
			sb.WriteString(`<h3 class="` + titleClass + `">` + template.HTMLEscapeString(title) + `</h3>`)
		}
		sb.WriteString(`<p class="` + textClass + `">` + ctx.SanitizeHTML(text) + `</p>`)
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</article>`)
	return sb.String(), nil
}
