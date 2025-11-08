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
	if len(courses) == 0 {
		return nil
	}

	cards := make([]courseCardTemplateData, 0, len(courses))

	for i := range courses {
		entry := courses[i]
		pkg := entry.Package
		access := entry.Access

		title := strings.TrimSpace(pkg.Title)
		if title == "" {
			title = "Untitled course"
		}

		element := "article"
		href := ""
		cardClass := "profile-course post-card"
		if pkg.ID > 0 {
			element = "a"
			href = fmt.Sprintf("/courses/%d", pkg.ID)
			cardClass += " profile-course--link"
		}

		headingID := fmt.Sprintf("profile-course-%d-title", i+1)
		descriptionID := ""
		descriptionHTML := template.HTML("")
		description := strings.TrimSpace(pkg.Description)
		if description != "" {
			descriptionID = fmt.Sprintf("profile-course-%d-description", i+1)
			descriptionHTML = template.HTML(template.HTMLEscapeString(description))
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

		card := courseCardTemplateData{
			Element:          element,
			Href:             href,
			CardClass:        cardClass,
			MediaClass:       "profile-course__media post-card__figure",
			ImageClass:       "profile-course__image post-card__image",
			ContentClass:     "profile-course__content post-card__content",
			TitleClass:       "profile-course__title post-card__title",
			MetaClass:        "profile-course__meta post-card__meta",
			DescriptionClass: "profile-course__description post-card__description",
			DescriptionTag:   "p",
			HeadingID:        headingID,
			DescriptionID:    descriptionID,
			Title:            title,
			MetaItems:        metaItems,
			Description:      descriptionHTML,
			Interactive:      false,
		}

		imageURL := strings.TrimSpace(pkg.ImageURL)
		if imageURL != "" {
			alt := "Course cover"
			if title != "" {
				alt = fmt.Sprintf("%s cover", title)
			}
			card.Image = &courseCardImage{URL: imageURL, Alt: alt}
		}

		cards = append(cards, card)
	}

	return cards
}
