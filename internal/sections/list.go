package sections

import (
	"fmt"
	"strings"

	"constructor-script-backend/internal/models"
)

// RegisterList registers the default list renderer on the provided registry.
func RegisterList(reg *Registry) {
	if reg == nil {
		return
	}
	reg.MustRegister("list", renderList)
}

func renderList(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)
	rawItems, ok := content["items"]
	if !ok {
		return "", nil
	}

	items := make([]string, 0)
	switch values := rawItems.(type) {
	case []interface{}:
		for _, item := range values {
			str, ok := item.(string)
			if !ok {
				continue
			}
			if trimmed := strings.TrimSpace(str); trimmed != "" {
				items = append(items, trimmed)
			}
		}
	case []string:
		for _, item := range values {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				items = append(items, trimmed)
			}
		}
	}

	if len(items) == 0 {
		return "", nil
	}

	ordered := false
	if rawOrdered, ok := content["ordered"]; ok {
		ordered = parseBool(rawOrdered, false)
	}

	listTag := "ul"
	listClass := fmt.Sprintf("%s__list", prefix)
	if ordered {
		listTag = "ol"
		listClass += " " + listClass + "--ordered"
	}

	itemClass := fmt.Sprintf("%s__list-item", prefix)
	sanitized := make([]string, len(items))
	for i, item := range items {
		sanitized[i] = ctx.SanitizeHTML(item)
	}

	var sb strings.Builder
	sb.WriteString(`<` + listTag + ` class="` + listClass + `">`)
	for _, item := range sanitized {
		sb.WriteString(`<li class="` + itemClass + `">` + item + `</li>`)
	}
	sb.WriteString(`</` + listTag + `>`)

	return sb.String(), nil
}
