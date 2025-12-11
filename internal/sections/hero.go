package sections

import (
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
)

// RegisterHero registers the hero section renderer.
func RegisterHero(reg *Registry) {
	if reg == nil {
		return
	}
	reg.RegisterSafe("hero", renderHero)
}

// RegisterHeroWithMetadata registers hero section with full metadata.
func RegisterHeroWithMetadata(reg *RegistryWithMetadata) {
	if reg == nil {
		return
	}

	desc := &SectionDescriptor{
		Renderer: renderHero,
		Metadata: SectionMetadata{
			Type:        "hero",
			Name:        "Hero Section",
			Description: "Displays a hero banner with title, subtitle, image and call-to-action button",
			Category:    "marketing",
			Icon:        "star",
			Schema: map[string]interface{}{
				"title": map[string]interface{}{
					"type":     "string",
					"required": true,
					"default":  "Welcome to Our Platform",
				},
				"subtitle": map[string]interface{}{
					"type":    "string",
					"default": "Discover amazing features and possibilities",
				},
				"text": map[string]interface{}{
					"type":    "string",
					"default": "",
				},
				"image_url": map[string]interface{}{
					"type":     "string",
					"required": true,
				},
				"image_alt": map[string]interface{}{
					"type":    "string",
					"default": "Hero image",
				},
				"button_text": map[string]interface{}{
					"type":    "string",
					"default": "Get started",
				},
				"button_url": map[string]interface{}{
					"type":     "string",
					"required": true,
					"default":  "/",
				},
			},
		},
	}

	reg.RegisterWithMetadata(desc)
}

func renderHero(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	// Hero section uses Settings from the Section model, not elements
	// The content is the entire section passed as interface{}
	var section models.Section

	switch v := elem.Content.(type) {
	case models.Section:
		section = v
	case *models.Section:
		if v != nil {
			section = *v
		}
	default:
		// Fallback: try to extract from element content map
		content := sectionContent(elem)
		if settings, ok := content["settings"].(map[string]interface{}); ok {
			section.Settings = settings
		} else {
			section.Settings = content
		}
	}

	// Extract hero settings
	settings := section.Settings
	if settings == nil {
		return "", nil
	}

	title, _ := settings["title"].(string)
	subtitle, _ := settings["subtitle"].(string)
	text, _ := settings["text"].(string)
	imageURL, _ := settings["image_url"].(string)
	imageAlt, _ := settings["image_alt"].(string)
	buttonText, _ := settings["button_text"].(string)
	buttonURL, _ := settings["button_url"].(string)

	// Validate required fields
	if strings.TrimSpace(title) == "" || strings.TrimSpace(imageURL) == "" {
		return "", nil
	}

	// Set default values
	if strings.TrimSpace(imageAlt) == "" {
		imageAlt = "Hero image"
	}
	if strings.TrimSpace(buttonText) == "" {
		buttonText = "Get started"
	}
	if strings.TrimSpace(buttonURL) == "" {
		buttonURL = "/"
	}

	// Sanitize HTML content
	sanitizedTitle := ctx.SanitizeHTML(title)
	sanitizedSubtitle := ctx.SanitizeHTML(subtitle)
	sanitizedText := ctx.SanitizeHTML(text)

	// Build CSS classes
	heroClass := fmt.Sprintf("%s__hero", prefix)
	heroContainerClass := fmt.Sprintf("%s__hero-container", prefix)
	heroContentClass := fmt.Sprintf("%s__hero-content", prefix)
	heroTitleClass := fmt.Sprintf("%s__hero-title", prefix)
	heroSubtitleClass := fmt.Sprintf("%s__hero-subtitle", prefix)
	heroTextClass := fmt.Sprintf("%s__hero-text", prefix)
	heroButtonClass := fmt.Sprintf("%s__hero-button", prefix)
	heroImageClass := fmt.Sprintf("%s__hero-image", prefix)
	heroImageImgClass := fmt.Sprintf("%s__hero-image-img", prefix)

	var sb strings.Builder
	sb.WriteString(`<div class="` + heroClass + `">`)
	sb.WriteString(`<div class="` + heroContainerClass + `">`)

	// Content section
	sb.WriteString(`<div class="` + heroContentClass + `">`)
	sb.WriteString(`<h1 class="` + heroTitleClass + `">` + sanitizedTitle + `</h1>`)

	if strings.TrimSpace(subtitle) != "" {
		sb.WriteString(`<h2 class="` + heroSubtitleClass + `">` + sanitizedSubtitle + `</h2>`)
	}

	if strings.TrimSpace(text) != "" {
		sb.WriteString(`<p class="` + heroTextClass + `">` + sanitizedText + `</p>`)
	}

	sb.WriteString(`<a href="` + template.HTMLEscapeString(buttonURL) + `" class="` + heroButtonClass + `">`)
	sb.WriteString(template.HTMLEscapeString(buttonText))
	sb.WriteString(`</a>`)
	sb.WriteString(`</div>`)

	// Image section
	sb.WriteString(`<div class="` + heroImageClass + `">`)
	sb.WriteString(`<img class="` + heroImageImgClass + `" src="` + template.HTMLEscapeString(imageURL) + `" alt="` + template.HTMLEscapeString(imageAlt) + `" />`)
	sb.WriteString(`</div>`)

	sb.WriteString(`</div>`)
	sb.WriteString(`</div>`)

	return sb.String(), nil
}
