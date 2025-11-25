package sections

import (
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
)

// RegisterFileGroup registers the default file group renderer on the provided registry.
func RegisterFileGroup(reg *Registry) {
	if reg == nil {
		return
	}
	reg.MustRegister("file_group", renderFileGroup)
}

func renderFileGroup(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)

	filesValue, ok := content["files"]
	if !ok {
		return "", nil
	}

	type fileEntry struct {
		url   string
		label string
	}

	files := make([]fileEntry, 0)
	switch items := filesValue.(type) {
	case []interface{}:
		for _, raw := range items {
			data, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			url, _ := data["url"].(string)
			label, _ := data["label"].(string)
			if url = strings.TrimSpace(url); url == "" {
				continue
			}
			files = append(files, fileEntry{
				url:   url,
				label: strings.TrimSpace(label),
			})
		}
	case []map[string]interface{}:
		for _, data := range items {
			url, _ := data["url"].(string)
			label, _ := data["label"].(string)
			if url = strings.TrimSpace(url); url == "" {
				continue
			}
			files = append(files, fileEntry{
				url:   url,
				label: strings.TrimSpace(label),
			})
		}
	}

	if len(files) == 0 {
		return "", nil
	}

	title := strings.TrimSpace(getString(content, "title"))
	description := strings.TrimSpace(getString(content, "description"))

	baseClass := fmt.Sprintf("%s__file-group", prefix)
	listClass := fmt.Sprintf("%s__file-list", prefix)
	itemClass := fmt.Sprintf("%s__file-list-item", prefix)
	linkClass := fmt.Sprintf("%s__file-link", prefix)
	titleClass := fmt.Sprintf("%s__file-group-title", prefix)
	descriptionClass := fmt.Sprintf("%s__file-group-description", prefix)

	var sb strings.Builder
	sb.WriteString(`<div class="` + baseClass + `">`)
	if title != "" {
		sb.WriteString(`<h3 class="` + titleClass + `">` + template.HTMLEscapeString(title) + `</h3>`)
	}
	if description != "" {
		sb.WriteString(`<p class="` + descriptionClass + `">` + ctx.SanitizeHTML(description) + `</p>`)
	}

	sb.WriteString(`<ul class="` + listClass + `">`)
	for _, file := range files {
		label := file.label
		if label == "" {
			label = file.url
		}
		sb.WriteString(`<li class="` + itemClass + `">`)
		sb.WriteString(`<a class="` + linkClass + `" href="` + template.HTMLEscapeString(file.url) + `" download>`) //nolint:lll
		sb.WriteString(template.HTMLEscapeString(label))
		sb.WriteString(`</a>`)
		sb.WriteString(`</li>`)
	}
	sb.WriteString(`</ul>`)
	sb.WriteString(`</div>`)

	return sb.String(), nil
}
