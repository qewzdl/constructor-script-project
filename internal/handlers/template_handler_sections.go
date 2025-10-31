package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/sections"
	"constructor-script-backend/pkg/logger"
)

func (h *TemplateHandler) renderSections(sections models.PostSections) (template.HTML, []string) {
	return h.renderSectionsWithPrefix(sections, "post")
}

func (h *TemplateHandler) renderSectionsWithPrefix(sections models.PostSections, prefix string) (template.HTML, []string) {
	if len(sections) == 0 {
		return "", nil
	}

	var sb strings.Builder
	var scripts []string

	for _, section := range sections {
		sectionType := strings.TrimSpace(strings.ToLower(section.Type))
		if sectionType == "" {
			sectionType = "standard"
		}

		title := strings.TrimSpace(section.Title)
		escapedTitle := template.HTMLEscapeString(title)

		baseClass := fmt.Sprintf("%s__section", prefix)
		sectionClasses := []string{baseClass, fmt.Sprintf("%s__section--%s", prefix, sectionType)}
		sectionTitleClass := fmt.Sprintf("%s__section-title", prefix)
		sectionImageWrapperClass := fmt.Sprintf("%s__section-image", prefix)
		sectionImageClass := fmt.Sprintf("%s__section-img", prefix)

		sb.WriteString(`<section class="` + strings.Join(sectionClasses, " ") + `" id="section-` + template.HTMLEscapeString(section.ID) + `">`)
		if title != "" {
			sb.WriteString(`<h2 class="` + sectionTitleClass + `">` + escapedTitle + `</h2>`)
		}

		if section.Image != "" {
			sb.WriteString(`<div class="` + sectionImageWrapperClass + `">`)
			sb.WriteString(`<img class="` + sectionImageClass + `" src="` + template.HTMLEscapeString(section.Image) + `" alt="` + escapedTitle + `" />`)
			sb.WriteString(`</div>`)
		}

		skipElements := false
		switch sectionType {
		case "hero":
			skipElements = true
		case "posts_list":
			skipElements = true
			sb.WriteString(h.renderPostsListSection(prefix, section))
		}

		if !skipElements {
			gridWrapperClass := ""
			gridItemClass := ""
			gridOpened := false
			if sectionType == "grid" {
				gridWrapperClass = fmt.Sprintf("%s__section-grid", prefix)
				gridItemClass = fmt.Sprintf("%s__section-grid-item", prefix)

				styleGridItems := true
				if section.StyleGridItems != nil {
					styleGridItems = *section.StyleGridItems
				}

				if !styleGridItems {
					gridItemClass = gridItemClass + " " + fmt.Sprintf("%s__section-grid-item--plain", prefix)
				}
			}

			for _, elem := range section.Elements {
				html, elemScripts := h.renderSectionElement(prefix, elem)
				scripts = appendScripts(scripts, elemScripts)
				if html == "" {
					continue
				}

				if sectionType == "grid" {
					if !gridOpened {
						sb.WriteString(`<div class="` + gridWrapperClass + `">`)
						gridOpened = true
					}
					sb.WriteString(`<div class="` + gridItemClass + `">` + html + `</div>`)
				} else {
					sb.WriteString(html)
				}
			}

			if gridOpened {
				sb.WriteString(`</div>`)
			}
		}

		sb.WriteString(`</section>`)
	}

	return template.HTML(sb.String()), scripts
}

func (h *TemplateHandler) renderPostsListSection(prefix string, section models.Section) string {
	listClass := fmt.Sprintf("%s__post-list", prefix)
	emptyClass := fmt.Sprintf("%s__post-list-empty", prefix)
	cardClass := fmt.Sprintf("%s__post-card post-card", prefix)

	limit := section.Limit
	if limit <= 0 {
		limit = constants.DefaultPostListSectionLimit
	}
	if limit > constants.MaxPostListSectionLimit {
		limit = constants.MaxPostListSectionLimit
	}

	if h.postService == nil {
		return `<p class="` + emptyClass + `">Posts are not available right now.</p>`
	}

	posts, err := h.postService.GetRecentPosts(limit)
	if err != nil {
		logger.Error(err, "Failed to load posts for section", map[string]interface{}{"section_id": section.ID})
		return `<p class="` + emptyClass + `">Unable to load posts at the moment. Please try again later.</p>`
	}

	if len(posts) == 0 {
		return `<p class="` + emptyClass + `">No posts available yet. Check back soon!</p>`
	}

	tmpl, err := h.templateClone()
	if err != nil {
		logger.Error(err, "Failed to clone templates for post list section", map[string]interface{}{"section_id": section.ID})
		return `<p class="` + emptyClass + `">Unable to display posts at the moment.</p>`
	}

	var sb strings.Builder
	sb.WriteString(`<div class="` + listClass + `">`)
	rendered := 0
	for i := range posts {
		post := posts[i]
		titleID := fmt.Sprintf("%s-post-%d", prefix, i+1)
		card, renderErr := h.renderPostCard(tmpl, &post, cardClass, titleID)
		if renderErr != nil {
			logger.Error(renderErr, "Failed to render post card", map[string]interface{}{"post_id": post.ID, "section_id": section.ID})
			continue
		}
		sb.WriteString(card)
		rendered++
	}
	sb.WriteString(`</div>`)

	if rendered == 0 {
		return `<p class="` + emptyClass + `">Unable to display posts at the moment.</p>`
	}

	return sb.String()
}

func (h *TemplateHandler) renderPostCard(tmpl *template.Template, post *models.Post, wrapperClass, titleID string) (string, error) {
	if tmpl == nil || post == nil {
		return "", fmt.Errorf("template or post is nil")
	}

	var buf bytes.Buffer
	data := map[string]interface{}{
		"Post":         post,
		"WrapperClass": wrapperClass,
		"TitleID":      titleID,
		"Heading":      "h3",
		"LazyLoad":     true,
	}

	if err := tmpl.ExecuteTemplate(&buf, "components/post-card", data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (h *TemplateHandler) renderSectionElement(prefix string, elem models.SectionElement) (string, []string) {
	if h == nil {
		return "", nil
	}

	if h.sectionRegistry == nil {
		h.sectionRegistry = sections.DefaultRegistry()
	}

	if renderer, ok := h.sectionRegistry.Get(elem.Type); ok {
		return renderer(h, prefix, elem)
	}

	return "", nil
}

func appendScripts(existing []string, additions []string) []string {
	if len(additions) == 0 {
		return existing
	}

	seen := make(map[string]struct{}, len(existing))
	for _, script := range existing {
		if script == "" {
			continue
		}
		seen[script] = struct{}{}
	}

	for _, script := range additions {
		if script == "" {
			continue
		}
		if _, ok := seen[script]; ok {
			continue
		}
		existing = append(existing, script)
		seen[script] = struct{}{}
	}

	return existing
}

func asScriptSlice(value interface{}) []string {
	if value == nil {
		return nil
	}

	if scripts, ok := value.([]string); ok {
		return scripts
	}

	if rawSlice, ok := value.([]interface{}); ok {
		result := make([]string, 0, len(rawSlice))
		for _, item := range rawSlice {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}

	return nil
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

		title := strings.TrimSpace(section.Title)
		if title == "" {
			continue
		}
		sb.WriteString(`<li class="post__toc-item">`)
		sb.WriteString(`<a href="#section-` + template.HTMLEscapeString(section.ID) + `" class="post__toc-link">`)
		sb.WriteString(template.HTMLEscapeString(title))
		sb.WriteString(`</a>`)
		sb.WriteString(`</li>`)
	}

	sb.WriteString(`</ol>`)
	sb.WriteString(`</nav>`)

	return template.HTML(sb.String())
}
