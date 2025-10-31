package sections

import (
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
)

// RegisterImageGroup registers the default image group renderer on the provided registry.
func RegisterImageGroup(reg *Registry) {
	if reg == nil {
		return
	}
	reg.MustRegister("image_group", renderImageGroup)
}

func renderImageGroup(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)
	layout, _ := content["layout"].(string)
	if layout == "" {
		layout = "grid"
	}

	groupClass := fmt.Sprintf("%s__image-group", prefix)
	wrapperClass := groupClass + " " + groupClass + "--" + template.HTMLEscapeString(layout)

	var sb strings.Builder
	sb.WriteString(`<div class="` + wrapperClass + `">`)

	if images, ok := content["images"].([]interface{}); ok {
		for _, raw := range images {
			imgMap, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}

			url, _ := imgMap["url"].(string)
			alt, _ := imgMap["alt"].(string)
			caption, _ := imgMap["caption"].(string)
			if strings.TrimSpace(url) == "" {
				continue
			}

			itemClass := fmt.Sprintf("%s__image-group-item", prefix)
			imgClass := fmt.Sprintf("%s__image-group-img", prefix)
			captionClass := fmt.Sprintf("%s__image-group-caption", prefix)

			sb.WriteString(`<figure class="` + itemClass + `">`)
			sb.WriteString(`<img class="` + imgClass + `" src="` + template.HTMLEscapeString(url) + `" alt="` + template.HTMLEscapeString(alt) + `" />`)
			if caption = strings.TrimSpace(caption); caption != "" {
				sb.WriteString(`<figcaption class="` + captionClass + `">` + ctx.SanitizeHTML(caption) + `</figcaption>`)
			}
			sb.WriteString(`</figure>`)
		}
	}

	sb.WriteString(`</div>`)
	return sb.String(), nil
}
