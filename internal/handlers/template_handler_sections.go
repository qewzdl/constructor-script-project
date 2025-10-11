package handlers

import (
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
)

func (h *TemplateHandler) renderSections(sections models.PostSections) template.HTML {
	if len(sections) == 0 {
		return ""
	}

	var sb strings.Builder

	for _, section := range sections {
		sb.WriteString(`<section class="post__section" id="section-` + template.HTMLEscapeString(section.ID) + `">`)
		sb.WriteString(`<h2 class="post__section-title">` + template.HTMLEscapeString(section.Title) + `</h2>`)

		if section.Image != "" {
			sb.WriteString(`<div class="post__section-image">`)
			sb.WriteString(`<img class="post__section-img" src="` + template.HTMLEscapeString(section.Image) + `" alt="` + template.HTMLEscapeString(section.Title) + `" />`)
			sb.WriteString(`</div>`)
		}

		for _, elem := range section.Elements {
			sb.WriteString(h.renderSectionElement(elem))
		}

		sb.WriteString(`</section>`)
	}

	return template.HTML(sb.String())
}

func (h *TemplateHandler) renderSectionElement(elem models.SectionElement) string {
	var sb strings.Builder

	contentMap, _ := elem.Content.(map[string]interface{})

	switch elem.Type {
	case "paragraph":
		if text, ok := contentMap["text"].(string); ok {
			sanitized := h.sanitizer.Sanitize(text)
			sb.WriteString(`<div class="post__paragraph">` + sanitized + `</div>`)
		}

	case "image":
		url, _ := contentMap["url"].(string)
		alt, _ := contentMap["alt"].(string)
		caption, _ := contentMap["caption"].(string)

		sb.WriteString(`<figure class="post__image">`)
		sb.WriteString(`<img class="post__image-img" src="` + template.HTMLEscapeString(url) + `" alt="` + template.HTMLEscapeString(alt) + `" />`)
		if caption != "" {
			sanitizedCaption := h.sanitizer.Sanitize(caption)
			sb.WriteString(`<figcaption class="post__image-caption">` + sanitizedCaption + `</figcaption>`)
		}
		sb.WriteString(`</figure>`)

	case "image_group":
		layout, _ := contentMap["layout"].(string)
		if layout == "" {
			layout = "grid"
		}
		sb.WriteString(`<div class="post__image-group post__image-group--` + template.HTMLEscapeString(layout) + `">`)

		if images, ok := contentMap["images"].([]interface{}); ok {
			for _, img := range images {
				if imgMap, ok := img.(map[string]interface{}); ok {
					url, _ := imgMap["url"].(string)
					alt, _ := imgMap["alt"].(string)
					caption, _ := imgMap["caption"].(string)

					sb.WriteString(`<figure class="post__image-group-item">`)
					sb.WriteString(`<img class="post__image-group-img" src="` + template.HTMLEscapeString(url) + `" alt="` + template.HTMLEscapeString(alt) + `" />`)
					if caption != "" {
						sanitizedCaption := h.sanitizer.Sanitize(caption)
						sb.WriteString(`<figcaption class="post__image-group-caption">` + sanitizedCaption + `</figcaption>`)
					}
					sb.WriteString(`</figure>`)
				}
			}
		}
		sb.WriteString(`</div>`)
	}

	return sb.String()
}

func (h *TemplateHandler) generateTOC(sections models.PostSections) template.HTML {
	if len(sections) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(`<nav class="post__toc" aria-label="Table of contents">`)
	sb.WriteString(`<h2 class="post__toc-title">Table of Contents</h2>`)
	sb.WriteString(`<ol class="post__toc-list">`)

	for _, section := range sections {
		sb.WriteString(`<li class="post__toc-item">`)
		sb.WriteString(`<a href="#section-` + template.HTMLEscapeString(section.ID) + `" class="post__toc-link">`)
		sb.WriteString(template.HTMLEscapeString(section.Title))
		sb.WriteString(`</a>`)
		sb.WriteString(`</li>`)
	}

	sb.WriteString(`</ol>`)
	sb.WriteString(`</nav>`)

	return template.HTML(sb.String())
}
