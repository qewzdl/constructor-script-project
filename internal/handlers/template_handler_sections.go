package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/pkg/logger"
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

		skipElements := false
		switch sectionType {
		case "hero":
			skipElements = true
		case "posts_list":
			skipElements = true
			sb.WriteString(h.renderPostsListSection(prefix, section))
		}

		if !skipElements {
			for _, elem := range section.Elements {
				sb.WriteString(h.renderSectionElement(prefix, elem))
			}
		}

		sb.WriteString(`</section>`)
	}

	return template.HTML(sb.String())
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
