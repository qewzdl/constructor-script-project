package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/pkg/logger"
)

type SectionRenderer func(h *TemplateHandler, prefix string, elem models.SectionElement) (string, []string)

func (h *TemplateHandler) RegisterSectionRenderer(sectionType string, renderer SectionRenderer) {
	if sectionType == "" || renderer == nil {
		return
	}

	if h.sectionRenderers == nil {
		h.sectionRenderers = make(map[string]SectionRenderer)
	}

	h.sectionRenderers[sectionType] = renderer
}

func (h *TemplateHandler) registerDefaultSectionRenderers() {
	if h.sectionRenderers == nil {
		h.sectionRenderers = make(map[string]SectionRenderer)
	}

	h.RegisterSectionRenderer("paragraph", renderParagraphSection)
	h.RegisterSectionRenderer("image", renderImageSection)
	h.RegisterSectionRenderer("image_group", renderImageGroupSection)
	h.RegisterSectionRenderer("list", renderListSection)
	h.RegisterSectionRenderer("search", renderSearchSection)
}

func renderParagraphSection(h *TemplateHandler, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)
	text, ok := content["text"].(string)
	if !ok || text == "" {
		return "", nil
	}

	var sb strings.Builder
	sanitized := h.sanitizer.Sanitize(text)
	paragraphClass := fmt.Sprintf("%s__paragraph", prefix)
	sb.WriteString(`<p class="` + paragraphClass + `">` + sanitized + `</p>`)
	return sb.String(), nil
}

func renderImageSection(h *TemplateHandler, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)
	url, _ := content["url"].(string)
	alt, _ := content["alt"].(string)
	caption, _ := content["caption"].(string)

	if url == "" {
		return "", nil
	}

	var sb strings.Builder
	figureClass := fmt.Sprintf("%s__image", prefix)
	imageClass := fmt.Sprintf("%s__image-img", prefix)
	sb.WriteString(`<figure class="` + figureClass + `">`)
	sb.WriteString(`<img class="` + imageClass + `" src="` + template.HTMLEscapeString(url) + `" alt="` + template.HTMLEscapeString(alt) + `" />`)
	if caption != "" {
		sanitizedCaption := h.sanitizer.Sanitize(caption)
		captionClass := fmt.Sprintf("%s__image-caption", prefix)
		sb.WriteString(`<figcaption class="` + captionClass + `">` + sanitizedCaption + `</figcaption>`)
	}
	sb.WriteString(`</figure>`)

	return sb.String(), nil
}

func renderImageGroupSection(h *TemplateHandler, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)
	layout, _ := content["layout"].(string)
	if layout == "" {
		layout = "grid"
	}

	var sb strings.Builder
	groupClass := fmt.Sprintf("%s__image-group", prefix)
	sb.WriteString(`<div class="` + groupClass + ` ` + groupClass + `--` + template.HTMLEscapeString(layout) + `">`)

	if images, ok := content["images"].([]interface{}); ok {
		for _, img := range images {
			imgMap, ok := img.(map[string]interface{})
			if !ok {
				continue
			}

			url, _ := imgMap["url"].(string)
			alt, _ := imgMap["alt"].(string)
			caption, _ := imgMap["caption"].(string)
			if url == "" {
				continue
			}

			itemClass := fmt.Sprintf("%s__image-group-item", prefix)
			imgClass := fmt.Sprintf("%s__image-group-img", prefix)
			captionClass := fmt.Sprintf("%s__image-group-caption", prefix)

			sb.WriteString(`<figure class="` + itemClass + `">`)
			sb.WriteString(`<img class="` + imgClass + `" src="` + template.HTMLEscapeString(url) + `" alt="` + template.HTMLEscapeString(alt) + `" />`)
			if caption != "" {
				sanitizedCaption := h.sanitizer.Sanitize(caption)
				sb.WriteString(`<figcaption class="` + captionClass + `">` + sanitizedCaption + `</figcaption>`)
			}
			sb.WriteString(`</figure>`)
		}
	}

	sb.WriteString(`</div>`)
	return sb.String(), nil
}

func renderListSection(h *TemplateHandler, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)

	rawItems, ok := content["items"]
	if !ok {
		return "", nil
	}

	var items []string

	switch values := rawItems.(type) {
	case []interface{}:
		for _, item := range values {
			str, ok := item.(string)
			if !ok {
				continue
			}
			str = strings.TrimSpace(str)
			if str == "" {
				continue
			}
			items = append(items, str)
		}
	case []string:
		for _, item := range values {
			str := strings.TrimSpace(item)
			if str == "" {
				continue
			}
			items = append(items, str)
		}
	}

	if len(items) == 0 {
		return "", nil
	}

	ordered := false

	if rawOrdered, ok := content["ordered"]; ok {
		switch value := rawOrdered.(type) {
		case bool:
			ordered = value
		case string:
			ordered = strings.EqualFold(value, "true")
		}
	}

	listTag := "ul"
	listClass := fmt.Sprintf("%s__list", prefix)
	if ordered {
		listTag = "ol"
		listClass += " " + listClass + "--ordered"
	}

	var sb strings.Builder
	sb.WriteString(`<` + listTag + ` class="` + listClass + `">`)

	for _, item := range items {
		sanitized := h.sanitizer.Sanitize(item)
		itemClass := fmt.Sprintf("%s__list-item", prefix)
		sb.WriteString(`<li class="` + itemClass + `">` + sanitized + `</li>`)
	}

	sb.WriteString(`</` + listTag + `>`)
	return sb.String(), nil
}

func renderSearchSection(h *TemplateHandler, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)

	title := strings.TrimSpace(getString(content, "title"))
	if title == "" {
		title = "Search"
	}

	description := strings.TrimSpace(getString(content, "description"))
	action := strings.TrimSpace(getString(content, "action"))
	if action == "" {
		action = "/search"
	}

	placeholder := strings.TrimSpace(getString(content, "placeholder"))
	if placeholder == "" {
		placeholder = "Start typing to search posts"
	}

	submitLabel := strings.TrimSpace(getString(content, "submit_label"))
	if submitLabel == "" {
		submitLabel = "Search"
	}

	filterLabel := strings.TrimSpace(getString(content, "filter_label"))
	if filterLabel == "" {
		filterLabel = "Filter by"
	}

	heading := normalizeHeading(getString(content, "heading"))
	if heading == "" {
		heading = "h2"
	}

	hint := strings.TrimSpace(getString(content, "hint"))
	if hint == "" {
		hint = "Use the search form above to explore the knowledge base."
	}

	showFilters := true
	if value, ok := content["show_filters"]; ok {
		showFilters = parseBool(value, true)
	}

	searchType := normalizeSearchType(getString(content, "default_type"))
	if searchType == "" {
		searchType = "all"
	}

	id := strings.TrimSpace(getString(content, "id"))
	if id == "" {
		id = fmt.Sprintf("%s-search-%s", prefix, elem.ID)
	}

	data := map[string]interface{}{
		"ID":          id,
		"Heading":     heading,
		"Title":       title,
		"Description": description,
		"Action":      action,
		"Placeholder": placeholder,
		"SubmitLabel": submitLabel,
		"FilterLabel": filterLabel,
		"ShowFilters": showFilters,
		"ShowResults": false,
		"Query":       "",
		"SearchType":  searchType,
		"HasQuery":    false,
		"Hint":        hint,
		"Result":      nil,
	}

	tmpl, err := h.templateClone()
	if err != nil {
		logger.Error(err, "Failed to clone templates for search section", map[string]interface{}{"element_id": elem.ID})
		return "", nil
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "components/search", data); err != nil {
		logger.Error(err, "Failed to render search section", map[string]interface{}{"element_id": elem.ID})
		return "", nil
	}

	return buf.String(), []string{"/static/js/search.js"}
}

func parseBool(value interface{}, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(v))
		if trimmed == "" {
			return fallback
		}
		switch trimmed {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		default:
			return fallback
		}
	default:
		return fallback
	}
}

func normalizeHeading(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	switch trimmed {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return trimmed
	default:
		return ""
	}
}

func normalizeSearchType(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	switch trimmed {
	case "title", "content", "tag", "author", "all":
		return trimmed
	default:
		return ""
	}
}

func sectionContent(elem models.SectionElement) map[string]interface{} {
	if contentMap, ok := elem.Content.(map[string]interface{}); ok {
		return contentMap
	}

	return map[string]interface{}{}
}
