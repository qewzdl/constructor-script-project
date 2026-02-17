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
	variation := strings.TrimSpace(strings.ToLower(section.Variation))
	if variation == "" && section.Settings != nil {
		if rawVariation, ok := section.Settings["variation"].(string); ok {
			variation = strings.TrimSpace(strings.ToLower(rawVariation))
		}
	}

	var items []string
	for _, item := range section.Elements {
		if strings.TrimSpace(strings.ToLower(item.Type)) != "feature_item" {
			continue
		}
		itemHTML, _ := renderFeatureItemWithVariation(ctx, prefix, item, variation)
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
	return renderFeatureItemWithVariation(ctx, prefix, elem, "")
}

func renderFeatureItemWithVariation(ctx RenderContext, prefix string, elem models.SectionElement, variation string) (string, []string) {
	content := sectionContent(elem)

	title := strings.TrimSpace(getString(content, "title"))
	subtitle := strings.TrimSpace(getString(content, "subtitle"))
	text := strings.TrimSpace(getString(content, "text"))
	imageURL := strings.TrimSpace(getString(content, "image_url"))
	imageAlt := strings.TrimSpace(getString(content, "image_alt"))
	normalisedVariation := strings.TrimSpace(strings.ToLower(variation))
	iconTextOnly := normalisedVariation == "glyph" || normalisedVariation == "icon-text"
	constellation := normalisedVariation == "constellation"

	if iconTextOnly {
		if title == "" {
			return "", nil
		}
	} else if constellation {
		if title == "" || subtitle == "" || text == "" || imageURL == "" {
			return "", nil
		}
	} else if text == "" {
		return "", nil
	}

	itemClass := fmt.Sprintf("%s__feature-item", prefix)
	headClass := fmt.Sprintf("%s__feature-head", prefix)
	metaClass := fmt.Sprintf("%s__feature-meta", prefix)
	bodyClass := fmt.Sprintf("%s__feature-body", prefix)
	mediaClass := fmt.Sprintf("%s__feature-media", prefix)
	imageClass := fmt.Sprintf("%s__feature-image", prefix)
	titleClass := fmt.Sprintf("%s__feature-title", prefix)
	subtitleClass := fmt.Sprintf("%s__feature-subtitle", prefix)
	textClass := fmt.Sprintf("%s__feature-text", prefix)

	var sb strings.Builder
	sb.WriteString(`<article class="` + itemClass + `">`)

	alt := imageAlt
	if alt == "" {
		if title != "" {
			alt = title
		} else if subtitle != "" {
			alt = subtitle
		} else {
			alt = text
		}
	}

	if constellation {
		sb.WriteString(`<div class="` + headClass + `">`)
		if imageURL != "" {
			sb.WriteString(`<div class="` + mediaClass + `">`)
			sb.WriteString(`<img class="` + imageClass + `" src="` + template.HTMLEscapeString(imageURL) + `" alt="` + template.HTMLEscapeString(alt) + `" />`)
			sb.WriteString(`</div>`)
		}
		sb.WriteString(`<div class="` + metaClass + `">`)
		sb.WriteString(`<h3 class="` + titleClass + `">` + template.HTMLEscapeString(title) + `</h3>`)
		sb.WriteString(`<p class="` + subtitleClass + `">` + template.HTMLEscapeString(subtitle) + `</p>`)
		sb.WriteString(`</div>`)
		sb.WriteString(`</div>`)
		sb.WriteString(`<p class="` + textClass + `">` + ctx.SanitizeHTML(text) + `</p>`)
		sb.WriteString(`</article>`)
		return sb.String(), nil
	}

	if imageURL != "" {
		sb.WriteString(`<div class="` + mediaClass + `">`)
		sb.WriteString(`<img class="` + imageClass + `" src="` + template.HTMLEscapeString(imageURL) + `" alt="` + template.HTMLEscapeString(alt) + `" />`)
		sb.WriteString(`</div>`)
	}

	hasBody := title != "" || (constellation && subtitle != "") || (!iconTextOnly && text != "")
	if hasBody {
		sb.WriteString(`<div class="` + bodyClass + `">`)
		if title != "" {
			sb.WriteString(`<h3 class="` + titleClass + `">` + template.HTMLEscapeString(title) + `</h3>`)
		}
		if constellation && subtitle != "" {
			sb.WriteString(`<p class="` + subtitleClass + `">` + template.HTMLEscapeString(subtitle) + `</p>`)
		}
		if !iconTextOnly && text != "" {
			sb.WriteString(`<p class="` + textClass + `">` + ctx.SanitizeHTML(text) + `</p>`)
		}
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</article>`)
	return sb.String(), nil
}
