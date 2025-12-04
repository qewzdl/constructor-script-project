package sections

import (
	"fmt"
	"strings"

	"constructor-script-backend/internal/models"
)

// RegisterParagraph registers the default paragraph renderer on the provided registry.
func RegisterParagraph(reg *Registry) {
	if reg == nil {
		return
	}
	reg.RegisterSafe("paragraph", renderParagraph)
}

func renderParagraph(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)
	text, ok := content["text"].(string)
	if !ok {
		return "", nil
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return "", nil
	}

	sanitized := ctx.SanitizeHTML(text)
	paragraphClass := fmt.Sprintf("%s__paragraph", prefix)
	return fmt.Sprintf(`<p class="%s">%s</p>`, paragraphClass, sanitized), nil
}
