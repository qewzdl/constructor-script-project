package sections

import (
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
)

// RegisterImage registers the default image renderer on the provided registry.
func RegisterImage(reg *Registry) {
	if reg == nil {
		return
	}
	reg.RegisterSafe("image", renderImage)
}

func renderImage(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)
	url, _ := content["url"].(string)
	alt, _ := content["alt"].(string)
	caption, _ := content["caption"].(string)

	if strings.TrimSpace(url) == "" {
		return "", nil
	}

	figureClass := fmt.Sprintf("%s__image", prefix)
	imageClass := fmt.Sprintf("%s__image-img", prefix)

	var sb strings.Builder
	sb.WriteString(`<figure class="` + figureClass + `">`)
	sb.WriteString(`<img class="` + imageClass + `" src="` + template.HTMLEscapeString(url) + `" alt="` + template.HTMLEscapeString(alt) + `" />`)
	if caption = strings.TrimSpace(caption); caption != "" {
		sanitizedCaption := ctx.SanitizeHTML(caption)
		captionClass := fmt.Sprintf("%s__image-caption", prefix)
		sb.WriteString(`<figcaption class="` + captionClass + `">` + sanitizedCaption + `</figcaption>`)
	}
	sb.WriteString(`</figure>`)

	return sb.String(), nil
}
