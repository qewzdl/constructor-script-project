package handlers

import (
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
)

type SectionRenderer func(h *TemplateHandler, prefix string, elem models.SectionElement) string

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
}

func renderParagraphSection(h *TemplateHandler, prefix string, elem models.SectionElement) string {
	content := sectionContent(elem)
	text, ok := content["text"].(string)
	if !ok || text == "" {
		return ""
	}

	var sb strings.Builder
	sanitized := h.sanitizer.Sanitize(text)
	paragraphClass := fmt.Sprintf("%s__paragraph", prefix)
	sb.WriteString(`<p class="` + paragraphClass + `">` + sanitized + `</p>`)
	return sb.String()
}

func renderImageSection(h *TemplateHandler, prefix string, elem models.SectionElement) string {
	content := sectionContent(elem)
	url, _ := content["url"].(string)
	alt, _ := content["alt"].(string)
	caption, _ := content["caption"].(string)

	if url == "" {
		return ""
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

	return sb.String()
}

func renderImageGroupSection(h *TemplateHandler, prefix string, elem models.SectionElement) string {
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
	return sb.String()
}

func renderListSection(h *TemplateHandler, prefix string, elem models.SectionElement) string {
	content := sectionContent(elem)

	rawItems, ok := content["items"]
	if !ok {
		return ""
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
		return ""
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
	return sb.String()
}

func sectionContent(elem models.SectionElement) map[string]interface{} {
	if contentMap, ok := elem.Content.(map[string]interface{}); ok {
		return contentMap
	}

	return map[string]interface{}{}
}
