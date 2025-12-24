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
	PriceBlock    *courseCardPriceBlock
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

type courseCardPriceBlock struct {
	Current       string
	CurrentClass  string
	Original      string
	OriginalClass string
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

type ownedCourseSectionData struct {
	Courses      []models.UserCoursePackage
	EmptyMessage string
}

func cloneUserCoursePackages(source []models.UserCoursePackage) []models.UserCoursePackage {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]models.UserCoursePackage, len(source))
	copy(cloned, source)
	return cloned
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
		slug := strings.TrimSpace(pkg.Slug)
		if slug != "" {
			element = "a"
			href = fmt.Sprintf("/courses/%s", slug)
			modifier = " profile-course--link"
			hasCourseID = pkg.ID > 0
			if pkg.ID > 0 {
				courseID = fmt.Sprintf("%d", pkg.ID)
			}
		} else if pkg.ID > 0 {
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
			Description:  strings.TrimSpace(pkg.Summary),
			Image:        image,
			MetaItems:    metaItems,
			HasCourseID:  hasCourseID,
			CourseID:     courseID,
		})
	}

	return entries
}
