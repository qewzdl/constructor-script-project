package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strconv"
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
			scripts = appendScripts(scripts, []string{"/static/js/courses-modal.js"})
			if h.courseCheckoutEnabled() {
				scripts = appendScripts(scripts, []string{"/static/js/courses-checkout.js"})
			}
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
	metaItemClass := fmt.Sprintf("%s__course-meta-item courses-list__meta-item", prefix)
	durationClass := fmt.Sprintf("%s__course-duration courses-list__duration", prefix)
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
		courseID := strconv.FormatUint(uint64(pkg.ID), 10)

		description := strings.TrimSpace(pkg.Description)
		sanitizedDescription := ""
		descriptionID := ""
		if description != "" {
			sanitizedDescription = strings.TrimSpace(h.SanitizeHTML(description))
			if sanitizedDescription != "" {
				descriptionID = fmt.Sprintf("%s-course-%d-description", prefix, rendered)
			}
		}

		articleAttrs := `<article class="` + cardClass + `" data-course-card`
		if courseID != "0" {
			articleAttrs += ` data-course-id="` + courseID + `"`
		}
		articleAttrs += ` role="button" tabindex="0" aria-haspopup="dialog" aria-labelledby="` + headingID + `"`
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

		metaItems := make([]string, 0, 4)

		priceLabel := formatCoursePrice(pkg.PriceCents)
		if priceLabel != "" {
			metaItems = append(metaItems, `<span class="`+priceClass+`">`+template.HTMLEscapeString(priceLabel)+`</span>`)
		}

		topicCount, lessonCount, totalDuration := coursePackageStats(pkg)

		topicLabel := formatTopicCount(topicCount)
		if topicLabel != "" {
			metaItems = append(metaItems, `<span class="`+metaItemClass+`">`+template.HTMLEscapeString(topicLabel)+`</span>`)
		}

		lessonLabel := formatLessonCount(lessonCount)
		if lessonLabel != "" {
			metaItems = append(metaItems, `<span class="`+metaItemClass+`">`+template.HTMLEscapeString(lessonLabel)+`</span>`)
		}

		durationLabel := formatVideoDuration(totalDuration)
		if durationLabel != "" {
			metaItems = append(metaItems, `<span class="`+metaItemClass+` `+durationClass+`">`+template.HTMLEscapeString(durationLabel)+`</span>`)
		}

		if len(metaItems) > 0 {
			sb.WriteString(`<div class="` + metaClass + `" aria-label="Course summary">`)
			for _, item := range metaItems {
				sb.WriteString(item)
			}
			sb.WriteString(`</div>`)
		}

		if description := strings.TrimSpace(pkg.Description); description != "" {
			sb.WriteString(`<div class="` + descriptionClass + `">` + h.SanitizeHTML(description) + `</div>`)
		}

		modalTopics := buildCourseModalTopics(pkg.Topics, h)

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
			if lessonCount := countTopicLessons(topic); lessonCount > 0 {
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

		if detailsJSON := buildCourseModalDetails(courseModalDetailsInput{
			Package:         pkg,
			Title:           title,
			DescriptionHTML: sanitizedDescription,
			ImageURL:        strings.TrimSpace(pkg.ImageURL),
			PriceLabel:      priceLabel,
			TopicLabel:      topicLabel,
			LessonLabel:     lessonLabel,
			DurationLabel:   durationLabel,
			Topics:          modalTopics,
			CourseID:        courseID,
		}); detailsJSON != "" {
			sb.WriteString(`<script type="application/json" data-course-details>`)
			sb.WriteString(detailsJSON)
			sb.WriteString(`</script>`)
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

type courseModalLesson struct {
	Title         string `json:"title,omitempty"`
	DurationLabel string `json:"duration_label,omitempty"`
}

type courseModalTopic struct {
	Title           string              `json:"title,omitempty"`
	DescriptionHTML string              `json:"description_html,omitempty"`
	LessonCount     int                 `json:"lesson_count,omitempty"`
	LessonLabel     string              `json:"lesson_label,omitempty"`
	DurationLabel   string              `json:"duration_label,omitempty"`
	Lessons         []courseModalLesson `json:"lessons,omitempty"`
}

type courseModalDetails struct {
	ID              string             `json:"id,omitempty"`
	Title           string             `json:"title,omitempty"`
	PriceText       string             `json:"price_text,omitempty"`
	DescriptionHTML string             `json:"description_html,omitempty"`
	ImageURL        string             `json:"image_url,omitempty"`
	ImageAlt        string             `json:"image_alt,omitempty"`
	Meta            []string           `json:"meta,omitempty"`
	Topics          []courseModalTopic `json:"topics,omitempty"`
}

type courseModalDetailsInput struct {
	Package         models.CoursePackage
	Title           string
	DescriptionHTML string
	ImageURL        string
	PriceLabel      string
	TopicLabel      string
	LessonLabel     string
	DurationLabel   string
	Topics          []courseModalTopic
	CourseID        string
}

func buildCourseModalTopics(topics []models.CourseTopic, h *TemplateHandler) []courseModalTopic {
	if len(topics) == 0 {
		return nil
	}

	result := make([]courseModalTopic, 0, len(topics))
	for _, topic := range topics {
		title := strings.TrimSpace(topic.Title)
		if title == "" {
			continue
		}

		descriptionHTML := ""
		if h != nil {
			if description := strings.TrimSpace(topic.Description); description != "" {
				sanitized := strings.TrimSpace(h.SanitizeHTML(description))
				if sanitized != "" {
					descriptionHTML = sanitized
				}
			}
		}

		lessonCount := countTopicLessons(topic)
		totalDuration := 0
		lessons := make([]courseModalLesson, 0, lessonCount)
		if len(topic.Steps) > 0 {
			for _, step := range topic.Steps {
				switch step.StepType {
				case models.CourseTopicStepTypeVideo:
					if step.Video == nil {
						continue
					}
					video := *step.Video
					if video.DurationSeconds > 0 {
						totalDuration += video.DurationSeconds
					}
					lessonTitle := strings.TrimSpace(video.Title)
					lesson := courseModalLesson{}
					if lessonTitle != "" {
						lesson.Title = lessonTitle
					}
					if durationLabel := formatVideoDuration(video.DurationSeconds); durationLabel != "" {
						lesson.DurationLabel = durationLabel
					}
					if lesson.Title != "" || lesson.DurationLabel != "" {
						lessons = append(lessons, lesson)
					}
				case models.CourseTopicStepTypeTest:
					lessonTitle := "Test"
					if step.Test != nil {
						name := strings.TrimSpace(step.Test.Title)
						if name != "" {
							lessonTitle = fmt.Sprintf("Test: %s", name)
						}
					}
					lessons = append(lessons, courseModalLesson{Title: lessonTitle})
				}
			}
		} else {
			for _, video := range topic.Videos {
				if video.DurationSeconds > 0 {
					totalDuration += video.DurationSeconds
				}
				lessonTitle := strings.TrimSpace(video.Title)
				lesson := courseModalLesson{}
				if lessonTitle != "" {
					lesson.Title = lessonTitle
				}
				if durationLabel := formatVideoDuration(video.DurationSeconds); durationLabel != "" {
					lesson.DurationLabel = durationLabel
				}
				if lesson.Title != "" || lesson.DurationLabel != "" {
					lessons = append(lessons, lesson)
				}
			}
		}

		topicData := courseModalTopic{
			Title:           title,
			DescriptionHTML: descriptionHTML,
		}

		if lessonCount > 0 {
			topicData.LessonCount = lessonCount
			if lessonLabel := formatLessonCount(lessonCount); lessonLabel != "" {
				topicData.LessonLabel = lessonLabel
			}
		}

		if totalDuration > 0 {
			if durationLabel := formatVideoDuration(totalDuration); durationLabel != "" {
				topicData.DurationLabel = durationLabel
			}
		}

		if len(lessons) > 0 {
			topicData.Lessons = lessons
		}

		result = append(result, topicData)
	}

	return result
}

func buildCourseModalDetails(input courseModalDetailsInput) string {
	title := strings.TrimSpace(input.Title)
	priceText := strings.TrimSpace(input.PriceLabel)
	descriptionHTML := strings.TrimSpace(input.DescriptionHTML)
	imageURL := strings.TrimSpace(input.ImageURL)

	details := courseModalDetails{
		ID:              input.CourseID,
		Title:           title,
		PriceText:       priceText,
		DescriptionHTML: descriptionHTML,
		Topics:          input.Topics,
	}

	if imageURL != "" {
		details.ImageURL = imageURL
		if title != "" {
			details.ImageAlt = fmt.Sprintf("%s course preview", title)
		}
	}

	meta := make([]string, 0, 3)
	if value := strings.TrimSpace(input.TopicLabel); value != "" {
		meta = append(meta, value)
	}
	if value := strings.TrimSpace(input.LessonLabel); value != "" {
		meta = append(meta, value)
	}
	if value := strings.TrimSpace(input.DurationLabel); value != "" {
		meta = append(meta, value)
	}
	if len(meta) > 0 {
		details.Meta = meta
	}

	jsonData, err := json.Marshal(details)
	if err != nil {
		logger.Error(err, "Failed to marshal course modal details", map[string]interface{}{"course_id": input.Package.ID})
		return ""
	}

	return string(jsonData)
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

func countTopicLessons(topic models.CourseTopic) int {
	if len(topic.Steps) == 0 {
		return len(topic.Videos)
	}
	count := 0
	for _, step := range topic.Steps {
		switch step.StepType {
		case models.CourseTopicStepTypeVideo,
			models.CourseTopicStepTypeTest:
			count++
		}
	}
	return count
}

func formatTopicCount(count int) string {
	if count <= 0 {
		return ""
	}
	if count == 1 {
		return "1 topic"
	}
	return fmt.Sprintf("%d topics", count)
}

func formatVideoDuration(totalSeconds int) string {
	if totalSeconds <= 0 {
		return ""
	}

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	parts := make([]string, 0, 3)
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if hours == 0 && minutes == 0 && seconds > 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, " ")
}

func coursePackageStats(pkg models.CoursePackage) (topics int, lessons int, duration int) {
	topics = len(pkg.Topics)
	if topics == 0 {
		return
	}

	seen := make(map[uint]struct{})
	for _, topic := range pkg.Topics {
		if len(topic.Steps) > 0 {
			for _, step := range topic.Steps {
				switch step.StepType {
				case models.CourseTopicStepTypeVideo:
					if step.Video == nil {
						continue
					}
					video := *step.Video
					if video.ID == 0 {
						duration += video.DurationSeconds
						lessons++
						continue
					}
					if _, ok := seen[video.ID]; ok {
						continue
					}
					seen[video.ID] = struct{}{}
					duration += video.DurationSeconds
					lessons++
				case models.CourseTopicStepTypeTest:
					lessons++
				}
			}
			continue
		}

		for _, video := range topic.Videos {
			if video.ID == 0 {
				duration += video.DurationSeconds
				lessons++
				continue
			}
			if _, ok := seen[video.ID]; ok {
				continue
			}
			seen[video.ID] = struct{}{}
			duration += video.DurationSeconds
			lessons++
		}
	}

	return
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
