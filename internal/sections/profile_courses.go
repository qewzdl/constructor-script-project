package sections

import (
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
)

const (
	profileCoursesDefaultTitle       = "Courses"
	profileCoursesDefaultDescription = "Review the learning packages currently available to your account."
	profileCoursesDefaultEmpty       = "You don't have any courses yet."
)

// RegisterProfileCourses registers the profile courses renderer with the registry.
func RegisterProfileCourses(reg *Registry) {
	if reg == nil {
		return
	}

	reg.RegisterSafe("profile_courses", renderProfileCourses)
}

type profileCourseEntry struct {
	Title       string
	Description string
	URL         string
	ImageURL    string
	ImageAlt    string
	CourseID    string
	Meta        []profileCourseMeta
}

type profileCourseMeta struct {
	Label    string
	Display  string
	DateTime string
}

func renderProfileCourses(_ RenderContext, _ string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)

	baseID := strings.TrimSpace(elem.ID)
	if baseID == "" {
		baseID = "profile-courses"
	}
	titleID := baseID + "-title"

	title := strings.TrimSpace(getString(content, "title"))
	if title == "" {
		title = profileCoursesDefaultTitle
	}

	description := strings.TrimSpace(getString(content, "description"))
	if description == "" {
		description = profileCoursesDefaultDescription
	}

	emptyMessage := strings.TrimSpace(getString(content, "empty_message"))
	if emptyMessage == "" {
		emptyMessage = profileCoursesDefaultEmpty
	}

	entries := parseProfileCourses(content["courses"])
	hasCourses := len(entries) > 0

	var sb strings.Builder
	sb.WriteString(`<section class="profile-card profile-card--courses" aria-labelledby="`)
	sb.WriteString(template.HTMLEscapeString(titleID))
	sb.WriteString(`">`)

	sb.WriteString(`<header class="profile-card__header">`)
	sb.WriteString(`<h2 id="` + template.HTMLEscapeString(titleID) + `" class="profile-card__title">`)
	sb.WriteString(template.HTMLEscapeString(title))
	sb.WriteString(`</h2>`)
	sb.WriteString(`<p class="profile-card__description">`)
	sb.WriteString(template.HTMLEscapeString(description))
	sb.WriteString(`</p>`)
	sb.WriteString(`</header>`)

	sb.WriteString(`<div class="profile-courses" id="` + template.HTMLEscapeString(baseID) + `" data-role="profile-courses">`)
	sb.WriteString(`<p class="profile-courses__empty"`)
	if hasCourses {
		sb.WriteString(` hidden`)
	}
	sb.WriteString(`>`)
	sb.WriteString(template.HTMLEscapeString(emptyMessage))
	sb.WriteString(`</p>`)

	sb.WriteString(`<ul class="profile-courses__list courses-list"`)
	if !hasCourses {
		sb.WriteString(` hidden`)
	}
	sb.WriteString(`>`)

	for index, course := range entries {
		headingID := fmt.Sprintf("%s-course-%d-title", baseID, index+1)
		descriptionID := ""
		if course.Description != "" {
			descriptionID = fmt.Sprintf("%s-course-%d-description", baseID, index+1)
		}

		sb.WriteString(`<li class="profile-courses__item">`)
		openTag := "article"
		if course.URL != "" {
			openTag = "a"
		}

		sb.WriteString(`<` + openTag)
		sb.WriteString(` class="profile-course post-card`)
		if course.URL != "" {
			sb.WriteString(` profile-course--link" href="` + template.HTMLEscapeString(course.URL) + `"`)
		} else {
			sb.WriteString(`"`)
		}
		sb.WriteString(` aria-labelledby="` + template.HTMLEscapeString(headingID) + `"`)
		if descriptionID != "" {
			sb.WriteString(` aria-describedby="` + template.HTMLEscapeString(descriptionID) + `"`)
		}
		if course.CourseID != "" {
			sb.WriteString(` data-course-id="` + template.HTMLEscapeString(course.CourseID) + `"`)
		}
		sb.WriteString(`>`)

		if course.ImageURL != "" {
			sb.WriteString(`<figure class="profile-course__media post-card__figure">`)
			sb.WriteString(`<img class="profile-course__image post-card__image" src="` + template.HTMLEscapeString(course.ImageURL) + `" alt="`)
			alt := course.ImageAlt
			if alt == "" {
				alt = "Course cover"
			}
			sb.WriteString(template.HTMLEscapeString(alt))
			sb.WriteString(`" loading="lazy" />`)
			sb.WriteString(`</figure>`)
		}

		sb.WriteString(`<div class="profile-course__content post-card__content">`)
		sb.WriteString(`<h3 id="` + template.HTMLEscapeString(headingID) + `" class="profile-course__title post-card__title">`)
		sb.WriteString(template.HTMLEscapeString(course.Title))
		sb.WriteString(`</h3>`)

		if course.Meta != nil {
			sb.WriteString(`<div class="profile-course__meta post-card__meta" aria-label="Course summary">`)
			for _, meta := range course.Meta {
				sb.WriteString(`<span class="profile-course__meta-item">`)
				if meta.Label != "" {
					sb.WriteString(template.HTMLEscapeString(meta.Label))
				}
				if meta.DateTime != "" {
					if meta.Label != "" {
						sb.WriteString(` `)
					}
					sb.WriteString(`<time datetime="` + template.HTMLEscapeString(meta.DateTime) + `">`)
					sb.WriteString(template.HTMLEscapeString(meta.Display))
					sb.WriteString(`</time>`)
				}
				sb.WriteString(`</span>`)
			}
			sb.WriteString(`</div>`)
		}

		if course.Description != "" {
			sb.WriteString(`<p class="profile-course__description post-card__description" id="` + template.HTMLEscapeString(descriptionID) + `">`)
			sb.WriteString(template.HTMLEscapeString(course.Description))
			sb.WriteString(`</p>`)
		}

		sb.WriteString(`</div>`)
		sb.WriteString(`</` + openTag + `>`)
		sb.WriteString(`</li>`)
	}

	sb.WriteString(`</ul>`)
	sb.WriteString(`</div>`)
	sb.WriteString(`</section>`)

	return sb.String(), nil
}

func parseProfileCourses(value interface{}) []profileCourseEntry {
	rawSlice, ok := value.([]map[string]interface{})
	if !ok {
		switch typed := value.(type) {
		case []interface{}:
			rawSlice = make([]map[string]interface{}, 0, len(typed))
			for _, item := range typed {
				if m, ok := item.(map[string]interface{}); ok {
					rawSlice = append(rawSlice, m)
				}
			}
		default:
			return nil
		}
	}

	entries := make([]profileCourseEntry, 0, len(rawSlice))
	for _, item := range rawSlice {
		if item == nil {
			continue
		}
		entry := profileCourseEntry{}
		if title, ok := item["title"].(string); ok {
			entry.Title = strings.TrimSpace(title)
		}
		if entry.Title == "" {
			entry.Title = "Untitled course"
		}
		if description, ok := item["description"].(string); ok {
			entry.Description = strings.TrimSpace(description)
		}
		if url, ok := item["url"].(string); ok {
			entry.URL = strings.TrimSpace(url)
		}
		if image, ok := item["image_url"].(string); ok {
			entry.ImageURL = strings.TrimSpace(image)
		}
		if alt, ok := item["image_alt"].(string); ok {
			entry.ImageAlt = strings.TrimSpace(alt)
		}
		if courseID, ok := item["course_id"].(string); ok {
			entry.CourseID = strings.TrimSpace(courseID)
		}

		if metaItems, ok := item["meta"].([]interface{}); ok {
			metas := make([]profileCourseMeta, 0, len(metaItems))
			for _, rawMeta := range metaItems {
				metaMap, ok := rawMeta.(map[string]interface{})
				if !ok {
					continue
				}
				meta := profileCourseMeta{}
				if label, ok := metaMap["label"].(string); ok {
					meta.Label = strings.TrimSpace(label)
				}
				if datetime, ok := metaMap["datetime"].(string); ok {
					meta.DateTime = strings.TrimSpace(datetime)
				}
				if display, ok := metaMap["display"].(string); ok {
					meta.Display = strings.TrimSpace(display)
				}
				if meta.Label == "" && meta.DateTime == "" && meta.Display == "" {
					continue
				}
				if meta.Display == "" {
					meta.Display = meta.DateTime
				}
				metas = append(metas, meta)
			}
			if len(metas) > 0 {
				entry.Meta = metas
			}
		}

		entries = append(entries, entry)
	}

	return entries
}
