package handlers

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"constructor-script-backend/internal/models"
)

type courseCardTemplateData struct {
	Element          string
	Href             string
	CardClass        string
	MediaClass       string
	ImageClass       string
	ContentClass     string
	TitleClass       string
	LinkClass        string
	MetaClass        string
	DescriptionClass string
	DescriptionTag   string
	TopicsClass      string
	TopicItemClass   string
	TopicNameClass   string
	TopicMetaClass   string

	HeadingID     string
	DescriptionID string
	HasCourseID   bool
	CourseID      string
	Title         string
	Image         *courseCardImage
	MetaItems     []courseCardMetaItem
	Description   template.HTML
	Topics        []courseCardTopic
	ModalDetails  template.JS
	Interactive   bool
}

type courseCardMetaItem struct {
	Class string
	Label string
	Time  *courseCardTime
}

type courseCardTime struct {
	DateTime string
	Display  string
}

type courseCardTopic struct {
	Name string
	Meta string
}

type courseCardImage struct {
	URL string
	Alt string
}

func formatCourseCardDate(t time.Time, format string) string {
	layouts := map[string]string{
		"short":    "01/02/2006",
		"medium":   "January 02, 2006",
		"long":     "Monday, January 02, 2006",
		"time":     "15:04",
		"datetime": "01/02/2006 15:04",
		"iso":      time.RFC3339,
	}

	if layout, ok := layouts[format]; ok {
		return t.Format(layout)
	}

	return t.Format(format)
}

func buildProfileCourseCards(courses []models.UserCoursePackage) []courseCardTemplateData {
	entries := buildProfileCourseEntries(courses)
	if len(entries) == 0 {
		return nil
	}

	cards := make([]courseCardTemplateData, 0, len(entries))
	for i := range entries {
		entry := entries[i]
		headingID := fmt.Sprintf("profile-course-%d-title", i+1)
		descriptionID := ""
		descriptionHTML := template.HTML("")
		if entry.Description != "" {
			descriptionID = fmt.Sprintf("profile-course-%d-description", i+1)
			descriptionHTML = template.HTML(template.HTMLEscapeString(entry.Description))
		}

		card := courseCardTemplateData{
			Element:          entry.Element,
			Href:             entry.Href,
			CardClass:        "profile-course post-card" + entry.CardModifier,
			MediaClass:       "profile-course__media post-card__figure",
			ImageClass:       "profile-course__image post-card__image",
			ContentClass:     "profile-course__content post-card__content",
			TitleClass:       "profile-course__title post-card__title",
			MetaClass:        "profile-course__meta post-card__meta",
			DescriptionClass: "profile-course__description post-card__description",
			DescriptionTag:   "p",
			HeadingID:        headingID,
			DescriptionID:    descriptionID,
			Title:            entry.Title,
			MetaItems:        entry.MetaItems,
			Description:      descriptionHTML,
			Interactive:      false,
			HasCourseID:      entry.HasCourseID,
			CourseID:         entry.CourseID,
		}

		if entry.Image != nil {
			card.Image = &courseCardImage{URL: entry.Image.URL, Alt: entry.Image.Alt}
		}

		cards = append(cards, card)
	}

	return cards
}

type profileCourseEntry struct {
	Element      string
	Href         string
	CardModifier string
	Title        string
	Description  string
	Image        *courseCardImage
	MetaItems    []courseCardMetaItem
	HasCourseID  bool
	CourseID     string
}

func buildProfileCourseEntries(courses []models.UserCoursePackage) []profileCourseEntry {
	if len(courses) == 0 {
		return nil
	}

	entries := make([]profileCourseEntry, 0, len(courses))
	for _, course := range courses {
		pkg := course.Package
		access := course.Access

		title := strings.TrimSpace(pkg.Title)
		if title == "" {
			title = "Untitled course"
		}

		element := "article"
		href := ""
		modifier := ""
		hasCourseID := false
		courseID := ""
		if pkg.ID > 0 {
			element = "a"
			href = fmt.Sprintf("/courses/%d", pkg.ID)
			modifier = " profile-course--link"
			hasCourseID = true
			courseID = fmt.Sprintf("%d", pkg.ID)
		}

		metaItems := make([]courseCardMetaItem, 0, 2)
		grantedDisplay := formatCourseCardDate(access.CreatedAt, "medium")
		grantedTime := &courseCardTime{
			DateTime: access.CreatedAt.Format(time.RFC3339),
			Display:  grantedDisplay,
		}
		metaItems = append(metaItems, courseCardMetaItem{
			Class: "profile-course__meta-item",
			Label: "Granted",
			Time:  grantedTime,
		})

		if access.ExpiresAt != nil {
			expiresDisplay := formatCourseCardDate(*access.ExpiresAt, "medium")
			metaItems = append(metaItems, courseCardMetaItem{
				Class: "profile-course__meta-item",
				Label: "Expires",
				Time: &courseCardTime{
					DateTime: access.ExpiresAt.Format(time.RFC3339),
					Display:  expiresDisplay,
				},
			})
		} else {
			metaItems = append(metaItems, courseCardMetaItem{
				Class: "profile-course__meta-item",
				Label: "No expiration",
			})
		}

		var image *courseCardImage
		if url := strings.TrimSpace(pkg.ImageURL); url != "" {
			alt := "Course cover"
			if title != "" {
				alt = fmt.Sprintf("%s cover", title)
			}
			image = &courseCardImage{URL: url, Alt: alt}
		}

		entries = append(entries, profileCourseEntry{
			Element:      element,
			Href:         href,
			CardModifier: modifier,
			Title:        title,
			Description:  strings.TrimSpace(pkg.Description),
			Image:        image,
			MetaItems:    metaItems,
			HasCourseID:  hasCourseID,
			CourseID:     courseID,
		})
	}

	return entries
}

func buildProfileCourseSectionContent(courses []models.UserCoursePackage) []map[string]interface{} {
	entries := buildProfileCourseEntries(courses)
	if len(entries) == 0 {
		return nil
	}

	result := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		data := map[string]interface{}{
			"title": entry.Title,
		}
		if entry.Description != "" {
			data["description"] = entry.Description
		}
		if entry.Href != "" {
			data["url"] = entry.Href
		}
		if entry.Image != nil {
			data["image_url"] = entry.Image.URL
			data["image_alt"] = entry.Image.Alt
		}
		if entry.HasCourseID {
			data["course_id"] = entry.CourseID
		}

		meta := make([]map[string]interface{}, 0, len(entry.MetaItems))
		for _, item := range entry.MetaItems {
			entryData := map[string]interface{}{}
			if trimmed := strings.TrimSpace(item.Label); trimmed != "" {
				entryData["label"] = trimmed
			}
			if item.Time != nil {
				if dt := strings.TrimSpace(item.Time.DateTime); dt != "" {
					entryData["datetime"] = dt
				}
				if disp := strings.TrimSpace(item.Time.Display); disp != "" {
					entryData["display"] = disp
				}
			}
			if len(entryData) > 0 {
				meta = append(meta, entryData)
			}
		}
		if len(meta) > 0 {
			data["meta"] = meta
		}

		result = append(result, data)
	}

	return result
}
