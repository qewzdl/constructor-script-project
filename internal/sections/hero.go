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
				"button_icon": map[string]interface{}{
					"type":    "string",
					"default": "",
				},
				"secondary_button_text": map[string]interface{}{
					"type":    "string",
					"default": "Learn more",
				},
				"secondary_button_url": map[string]interface{}{
					"type":    "string",
					"default": "/",
				},
				"secondary_button_icon": map[string]interface{}{
					"type":    "string",
					"default": "",
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
	buttonIcon, _ := settings["button_icon"].(string)
	secondaryButtonText, _ := settings["secondary_button_text"].(string)
	secondaryButtonURL, _ := settings["secondary_button_url"].(string)
	secondaryButtonIcon, _ := settings["secondary_button_icon"].(string)
	variation := strings.TrimSpace(strings.ToLower(section.Variation))
	if variation == "" {
		if rawVariation, ok := settings["variation"].(string); ok {
			variation = strings.TrimSpace(strings.ToLower(rawVariation))
		}
	}
	if variation == "" {
		variation = "split"
	}
	switch variation {
	case "split", "centered", "minimal", "immersive":
	default:
		variation = "split"
	}

	// Validate required fields
	if strings.TrimSpace(title) == "" {
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
	if strings.TrimSpace(secondaryButtonText) == "" {
		secondaryButtonText = "Learn more"
	}
	if strings.TrimSpace(secondaryButtonURL) == "" {
		secondaryButtonURL = "/"
	}

	// Sanitize HTML content
	sanitizedTitle := ctx.SanitizeHTML(title)
	sanitizedSubtitle := ctx.SanitizeHTML(subtitle)
	sanitizedText := ctx.SanitizeHTML(text)

	// Build CSS classes
	heroClass := fmt.Sprintf("%s__hero", prefix)
	if variation != "" {
		heroClass = heroClass + " " + fmt.Sprintf("%s__hero--variation-%s", prefix, template.HTMLEscapeString(variation))
	}
	heroContainerClass := fmt.Sprintf("%s__hero-container", prefix)
	heroContainerCenteredClass := fmt.Sprintf("%s__hero-container--centered", prefix)
	heroContainerImmersiveClass := fmt.Sprintf("%s__hero-container--immersive", prefix)
	heroContentClass := fmt.Sprintf("%s__hero-content", prefix)
	heroContentImmersiveClass := fmt.Sprintf("%s__hero-content--immersive", prefix)
	heroTitleClass := fmt.Sprintf("%s__hero-title", prefix)
	heroSubtitleClass := fmt.Sprintf("%s__hero-subtitle", prefix)
	heroTextClass := fmt.Sprintf("%s__hero-text", prefix)
	heroActionsClass := fmt.Sprintf("%s__hero-actions", prefix)
	heroButtonClass := fmt.Sprintf("%s__hero-button", prefix)
	heroButtonPrimaryClass := fmt.Sprintf("%s__hero-button--primary", prefix)
	heroButtonSecondaryClass := fmt.Sprintf("%s__hero-button--secondary", prefix)
	heroButtonIconClass := fmt.Sprintf("%s__hero-button-icon", prefix)
	heroStageClass := fmt.Sprintf("%s__hero-stage", prefix)
	heroOverlayClass := fmt.Sprintf("%s__hero-overlay", prefix)
	heroImageClass := fmt.Sprintf("%s__hero-image", prefix)
	heroImageCenteredClass := fmt.Sprintf("%s__hero-image--centered", prefix)
	heroImageImgClass := fmt.Sprintf("%s__hero-image-img", prefix)

	var sb strings.Builder
	renderContent := func(builder *strings.Builder, variationClass string, immersiveLayout bool) {
		contentClass := heroContentClass
		if variationClass != "" {
			contentClass = contentClass + ` ` + variationClass
		}
		primaryIcon := strings.TrimSpace(buttonIcon)
		secondaryIcon := strings.TrimSpace(secondaryButtonIcon)

		builder.WriteString(`<div class="` + contentClass + `">`)
		if immersiveLayout {
			if strings.TrimSpace(subtitle) != "" {
				builder.WriteString(`<h2 class="` + heroSubtitleClass + `">` + sanitizedSubtitle + `</h2>`)
			}
			builder.WriteString(`<h1 class="` + heroTitleClass + `">` + sanitizedTitle + `</h1>`)

			if strings.TrimSpace(text) != "" {
				builder.WriteString(`<p class="` + heroTextClass + `">` + sanitizedText + `</p>`)
			}

			builder.WriteString(`<div class="` + heroActionsClass + `">`)
			builder.WriteString(`<a href="` + template.HTMLEscapeString(buttonURL) + `" class="` + heroButtonClass + ` ` + heroButtonPrimaryClass + `">`)
			if primaryIcon != "" {
				builder.WriteString(`<span class="` + heroButtonIconClass + `" aria-hidden="true">` + template.HTMLEscapeString(primaryIcon) + `</span>`)
			}
			builder.WriteString(template.HTMLEscapeString(buttonText))
			builder.WriteString(`</a>`)
			builder.WriteString(`<a href="` + template.HTMLEscapeString(secondaryButtonURL) + `" class="` + heroButtonClass + ` ` + heroButtonSecondaryClass + `">`)
			if secondaryIcon != "" {
				builder.WriteString(`<span class="` + heroButtonIconClass + `" aria-hidden="true">` + template.HTMLEscapeString(secondaryIcon) + `</span>`)
			}
			builder.WriteString(template.HTMLEscapeString(secondaryButtonText))
			builder.WriteString(`</a>`)
			builder.WriteString(`</div>`)
			builder.WriteString(`</div>`)
			return
		}

		builder.WriteString(`<h1 class="` + heroTitleClass + `">` + sanitizedTitle + `</h1>`)

		if strings.TrimSpace(subtitle) != "" {
			builder.WriteString(`<h2 class="` + heroSubtitleClass + `">` + sanitizedSubtitle + `</h2>`)
		}

		if strings.TrimSpace(text) != "" {
			builder.WriteString(`<p class="` + heroTextClass + `">` + sanitizedText + `</p>`)
		}

		builder.WriteString(`<a href="` + template.HTMLEscapeString(buttonURL) + `" class="` + heroButtonClass + `">`)
		builder.WriteString(template.HTMLEscapeString(buttonText))
		builder.WriteString(`</a>`)
		builder.WriteString(`</div>`)
	}

	imageURL = strings.TrimSpace(imageURL)
	sb.WriteString(`<div class="` + heroClass + `">`)
	if variation == "immersive" {
		sb.WriteString(`<div class="` + heroStageClass + `">`)
		sb.WriteString(`<div class="` + heroOverlayClass + `" aria-hidden="true"></div>`)
		sb.WriteString(`<div class="` + heroContainerClass + ` ` + heroContainerImmersiveClass + `">`)
		renderContent(&sb, heroContentImmersiveClass, true)
		sb.WriteString(`</div>`)
		sb.WriteString(`</div>`)
		sb.WriteString(`</div>`)
		return sb.String(), []string{"/static/js/hero-immersive-cursor.js"}
	}

	if variation == "centered" {
		sb.WriteString(`<div class="` + heroContainerClass + ` ` + heroContainerCenteredClass + `">`)
		renderContent(&sb, "", false)
		sb.WriteString(`</div>`)
		if imageURL != "" {
			sb.WriteString(`<figure class="` + heroImageClass + ` ` + heroImageCenteredClass + `">`)
			sb.WriteString(`<img class="` + heroImageImgClass + `" src="` + template.HTMLEscapeString(imageURL) + `" alt="` + template.HTMLEscapeString(imageAlt) + `" />`)
			sb.WriteString(`</figure>`)
		}
		sb.WriteString(`</div>`)
		return sb.String(), nil
	}

	sb.WriteString(`<div class="` + heroContainerClass + `">`)
	renderContent(&sb, "", false)
	if imageURL != "" && variation != "minimal" {
		sb.WriteString(`<div class="` + heroImageClass + `">`)
		sb.WriteString(`<img class="` + heroImageImgClass + `" src="` + template.HTMLEscapeString(imageURL) + `" alt="` + template.HTMLEscapeString(imageAlt) + `" />`)
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</div>`)
	sb.WriteString(`</div>`)

	return sb.String(), nil
}
