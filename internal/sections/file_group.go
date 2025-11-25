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
	iconClass := fmt.Sprintf("%s__file-link-icon", prefix)
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
		sb.WriteString(`<svg class="` + iconClass + `" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><g stroke-width="0"></g><g stroke-linecap="round" stroke-linejoin="round"></g><g> <g id="style=stroke"> <g id="attach"> <path id="vector (Stroke)" fill-rule="evenodd" clip-rule="evenodd" d="M17.3656 4.70536C16.2916 3.63142 14.5504 3.63142 13.4765 4.70536L5.34477 12.8371C3.68504 14.4968 3.68504 17.1878 5.34477 18.8475C7.0045 20.5072 9.69545 20.5072 11.3552 18.8475L20.1172 10.0855C20.41 9.79263 20.8849 9.79263 21.1778 10.0855C21.4707 10.3784 21.4707 10.8533 21.1778 11.1462L12.4158 19.9082C10.1703 22.1537 6.52963 22.1537 4.28411 19.9082C2.0386 17.6626 2.0386 14.0219 4.28411 11.7764L12.4158 3.6447C14.0756 1.98497 16.7665 1.98497 18.4262 3.6447C20.086 5.30443 20.086 7.99538 18.4262 9.65511L10.6327 17.4487C9.55876 18.5226 7.81756 18.5226 6.74361 17.4487C5.66967 16.3747 5.66967 14.6335 6.74361 13.5596L13.9377 6.36552C14.2305 6.07263 14.7054 6.07263 14.9983 6.36552C15.2912 6.65842 15.2912 7.13329 14.9983 7.42618L7.80427 14.6202C7.31612 15.1084 7.31612 15.8998 7.80427 16.388C8.29243 16.8761 9.08389 16.8761 9.57204 16.388L17.3656 8.59445C18.4395 7.5205 18.4395 5.7793 17.3656 4.70536Z"></path> </g> </g> </g></svg>`)
		sb.WriteString(`<span>` + template.HTMLEscapeString(label) + `</span>`)
		sb.WriteString(`</a>`)
		sb.WriteString(`</li>`)
	}
	sb.WriteString(`</ul>`)
	sb.WriteString(`</div>`)

	return sb.String(), nil
}
