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

		if (sectionType == "posts_list" || sectionType == "categories_list") && !h.blogEnabled() {
			continue
		}

		if sectionType == "courses_list" && !h.coursesEnabled() {
			continue
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
		case "categories_list":
			skipElements = true
			sb.WriteString(h.renderCategoriesListSection(prefix, section))
		case "courses_list":
			skipElements = true
			sb.WriteString(h.renderCoursesListSection(prefix, section))
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

func (h *TemplateHandler) renderCategoriesListSection(prefix string, section models.Section) string {
	emptyClass := fmt.Sprintf("%s__category-list-empty blog__empty", prefix)

	limit := section.Limit
	if limit <= 0 {
		limit = constants.DefaultCategoryListSectionLimit
	}
	if limit > constants.MaxCategoryListSectionLimit {
		limit = constants.MaxCategoryListSectionLimit
	}

	if h.categoryService == nil {
		return `<p class="` + emptyClass + `">Categories are not available right now.</p>`
	}

	categories, err := h.categoryService.GetAll()
	if err != nil {
		logger.Error(err, "Failed to load categories for section", map[string]interface{}{"section_id": section.ID})
		return `<p class="` + emptyClass + `">Unable to load categories at the moment. Please try again later.</p>`
	}

	filtered := make([]models.Category, 0, len(categories))
	for _, category := range categories {
		if strings.EqualFold(category.Slug, "uncategorized") || strings.EqualFold(category.Name, "uncategorized") {
			continue
		}
		filtered = append(filtered, category)
	}

	if len(filtered) == 0 {
		return `<p class="` + emptyClass + `">No categories available yet. Check back soon!</p>`
	}

	if limit < len(filtered) {
		filtered = filtered[:limit]
	}

	navClass := fmt.Sprintf("%s__categories blog__categories", prefix)

	var sb strings.Builder
	sb.WriteString(`<nav class="` + navClass + `" aria-label="Browse by category">`)
	sb.WriteString(`<ul class="category-list">`)
	for _, category := range filtered {
		slug := template.HTMLEscapeString(category.Slug)
		name := strings.TrimSpace(category.Name)
		if name == "" {
			name = category.Slug
		}
		sb.WriteString(`<li class="category-list__item">`)
		sb.WriteString(`<a class="category-list__link" href="/category/` + slug + `">`)
		sb.WriteString(template.HTMLEscapeString(name))
		sb.WriteString(`</a>`)
		sb.WriteString(`</li>`)
	}
	sb.WriteString(`</ul>`)
	sb.WriteString(`</nav>`)

	return sb.String()
}

func (h *TemplateHandler) renderCoursesListSection(prefix string, section models.Section) string {
	const maxTopicsPerCourse = 6

	listClass := fmt.Sprintf("%s__course-list courses-list", prefix)
	emptyClass := fmt.Sprintf("%s__course-list-empty courses-list__empty", prefix)
	cardClass := fmt.Sprintf("%s__course-card courses-list__item post-card", prefix)
	mediaClass := fmt.Sprintf("%s__course-media post-card__figure", prefix)
	imageClass := fmt.Sprintf("%s__course-image post-card__image", prefix)
	contentClass := fmt.Sprintf("%s__course-content post-card__content", prefix)
	titleClass := fmt.Sprintf("%s__course-title post-card__title", prefix)
	linkClass := fmt.Sprintf("%s__course-link post-card__link post-card__link--static", prefix)
	priceClass := fmt.Sprintf("%s__course-price courses-list__price", prefix)
	metaClass := fmt.Sprintf("%s__course-meta post-card__meta", prefix)
	descriptionClass := fmt.Sprintf("%s__course-description post-card__description", prefix)
	topicsClass := fmt.Sprintf("%s__course-topics post-card__tags courses-list__topics", prefix)
	topicItemClass := fmt.Sprintf("%s__course-topic post-card__tag", prefix)
	topicNameClass := fmt.Sprintf("%s__course-topic-name post-card__tag-link post-card__tag-link--static", prefix)
	topicMetaClass := fmt.Sprintf("%s__course-topic-meta courses-list__topic-meta", prefix)

	limit := section.Limit
	if limit <= 0 {
		limit = constants.DefaultCourseListSectionLimit
	}
	if limit > constants.MaxCourseListSectionLimit {
		limit = constants.MaxCourseListSectionLimit
	}

	if h == nil || h.coursePackageSvc == nil {
		return `<p class="` + emptyClass + `">Courses are not available right now.</p>`
	}

	packages, err := h.coursePackageSvc.List()
	if err != nil {
		logger.Error(err, "Failed to load course packages for section", map[string]interface{}{"section_id": section.ID})
		return `<p class="` + emptyClass + `">Unable to load courses at the moment. Please try again later.</p>`
	}

	if len(packages) == 0 {
		return `<p class="` + emptyClass + `">No courses available yet. Check back soon!</p>`
	}

	if limit < len(packages) {
		packages = packages[:limit]
	}

	var sb strings.Builder
	sb.WriteString(`<div class="` + listClass + `">`)

	rendered := 0
	for i := range packages {
		pkg := packages[i]
		title := strings.TrimSpace(pkg.Title)
		if title == "" {
			continue
		}

		rendered++
		headingID := fmt.Sprintf("%s-course-%d-title", prefix, rendered)

		description := strings.TrimSpace(pkg.Description)
		sanitizedDescription := ""
		descriptionID := ""
		if description != "" {
			sanitizedDescription = strings.TrimSpace(h.SanitizeHTML(description))
			if sanitizedDescription != "" {
				descriptionID = fmt.Sprintf("%s-course-%d-description", prefix, rendered)
			}
		}

		articleAttrs := `<article class="` + cardClass + `" aria-labelledby="` + headingID + `"`
		if descriptionID != "" {
			articleAttrs += ` aria-describedby="` + descriptionID + `"`
		}
		articleAttrs += `>`
		sb.WriteString(articleAttrs)

		if image := strings.TrimSpace(pkg.ImageURL); image != "" {
			sb.WriteString(`<figure class="` + mediaClass + `">`)
			sb.WriteString(`<img class="` + imageClass + `" src="` + template.HTMLEscapeString(image) + `" alt="` + template.HTMLEscapeString(title) + ` course preview" loading="lazy" />`)
			sb.WriteString(`</figure>`)
		}

		sb.WriteString(`<div class="` + contentClass + `">`)
		sb.WriteString(`<h3 id="` + headingID + `" class="` + titleClass + `">`)
		sb.WriteString(`<span class="` + linkClass + `">` + template.HTMLEscapeString(title) + `</span>`)
		sb.WriteString(`</h3>`)

		if price := formatCoursePrice(pkg.PriceCents); price != "" {
			sb.WriteString(`<div class="` + metaClass + `">`)
			sb.WriteString(`<span class="` + priceClass + `">` + template.HTMLEscapeString(price) + `</span>`)
			sb.WriteString(`</div>`)
		}

		if description := strings.TrimSpace(pkg.Description); description != "" {
			sb.WriteString(`<div class="` + descriptionClass + `">` + h.SanitizeHTML(description) + `</div>`)
		}

		topicsRendered := 0
		for _, topic := range pkg.Topics {
			if topicsRendered >= maxTopicsPerCourse {
				break
			}
			name := strings.TrimSpace(topic.Title)
			if name == "" {
				continue
			}
			if topicsRendered == 0 {
				sb.WriteString(`<ul class="` + topicsClass + `" aria-label="Included topics">`)
			}
			sb.WriteString(`<li class="` + topicItemClass + `">`)
			sb.WriteString(`<span class="` + topicNameClass + `">` + template.HTMLEscapeString(name))
			if lessonCount := len(topic.Videos); lessonCount > 0 {
				lessonLabel := formatLessonCount(lessonCount)
				if lessonLabel != "" {
					sb.WriteString(` <span class="` + topicMetaClass + `">` + template.HTMLEscapeString(lessonLabel) + `</span>`)
				}
			}
			sb.WriteString(`</span>`)
			sb.WriteString(`</li>`)
			topicsRendered++
		}

		if topicsRendered > 0 {
			sb.WriteString(`</ul>`)
		}

		sb.WriteString(`</div>`)
		sb.WriteString(`</article>`)
	}

	sb.WriteString(`</div>`)

	if rendered == 0 {
		return `<p class="` + emptyClass + `">No courses available yet. Check back soon!</p>`
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

func formatCoursePrice(priceCents int64) string {
	if priceCents <= 0 {
		return "Free"
	}

	dollars := priceCents / 100
	cents := priceCents % 100

	if cents == 0 {
		return fmt.Sprintf("$%d", dollars)
	}

	return fmt.Sprintf("$%d.%02d", dollars, cents)
}

func formatLessonCount(count int) string {
	if count <= 0 {
		return ""
	}
	if count == 1 {
		return "1 lesson"
	}
	return fmt.Sprintf("%d lessons", count)
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
