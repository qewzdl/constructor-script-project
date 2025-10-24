package handlers

import (
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
)

func (h *TemplateHandler) renderSections(sections models.PostSections) template.HTML {
	return h.renderSectionsWithPrefix(sections, "post")
}

func (h *TemplateHandler) renderSectionsWithPrefix(sections models.PostSections, prefix string) template.HTML {
	if len(sections) == 0 {
		return ""
	}

	var sb strings.Builder

	for _, section := range sections {
		sectionType := strings.TrimSpace(strings.ToLower(section.Type))
		if sectionType == "" {
			sectionType = "standard"
		}

		baseClass := fmt.Sprintf("%s__section", prefix)
		sectionClasses := []string{baseClass, fmt.Sprintf("%s__section--%s", prefix, sectionType)}
		sectionTitleClass := fmt.Sprintf("%s__section-title", prefix)
		sectionImageWrapperClass := fmt.Sprintf("%s__section-image", prefix)
		sectionImageClass := fmt.Sprintf("%s__section-img", prefix)

		sb.WriteString(`<section class="` + strings.Join(sectionClasses, " ") + `" id="section-` + template.HTMLEscapeString(section.ID) + `">`)
		sb.WriteString(`<h2 class="` + sectionTitleClass + `">` + template.HTMLEscapeString(section.Title) + `</h2>`)

		if section.Image != "" {
			sb.WriteString(`<div class="` + sectionImageWrapperClass + `">`)
			sb.WriteString(`<img class="` + sectionImageClass + `" src="` + template.HTMLEscapeString(section.Image) + `" alt="` + template.HTMLEscapeString(section.Title) + `" />`)
			sb.WriteString(`</div>`)
		}

		if sectionType != "hero" {
			for _, elem := range section.Elements {
				sb.WriteString(h.renderSectionElement(prefix, elem))
			}
		}

		sb.WriteString(`</section>`)
	}

	return template.HTML(sb.String())
}

func (h *TemplateHandler) renderSectionElement(prefix string, elem models.SectionElement) string {
	if h.sectionRenderers == nil {
		return ""
	}

	if renderer, ok := h.sectionRenderers[elem.Type]; ok {
		return renderer(h, prefix, elem)
	}

	return ""
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
		if strings.EqualFold(section.Type, "hero") {
			continue
		}
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
