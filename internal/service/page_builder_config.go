package service

import (
	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
)

// GetPageBuilderConfig returns configuration for the page builder UI.
func (s *PageService) GetPageBuilderConfig() models.PageBuilderConfig {
	return models.PageBuilderConfig{
		AvailableSections: getAvailableSectionTypes(),
		DefaultPadding:    constants.DefaultSectionPadding,
		DefaultMargin:     constants.DefaultSectionMargin,
		PaddingOptions:    constants.SectionPaddingOptions(),
		MarginOptions:     constants.SectionMarginOptions(),
	}
}

func getAvailableSectionTypes() []models.SectionTypeConfig {
	return []models.SectionTypeConfig{
		{
			Type:        "standard",
			Name:        "Standard Section",
			Description: "Basic content section with custom elements",
			Category:    "layout",
			Icon:        "layout",
			AllowedIn:   []string{"page", "post", "homepage"},
			Schema: map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"label":       "Section Title",
					"placeholder": "Enter section title",
				},
				"padding_vertical": map[string]interface{}{
					"type":  "number",
					"label": "Vertical Padding",
				},
			},
		},
		{
			Type:        "grid",
			Name:        "Grid Section",
			Description: "Display content in a grid layout",
			Category:    "layout",
			Icon:        "grid",
			AllowedIn:   []string{"page", "post", "homepage"},
			Schema: map[string]interface{}{
				"title": map[string]interface{}{
					"type":  "string",
					"label": "Section Title",
				},
				"style_grid_items": map[string]interface{}{
					"type":    "boolean",
					"label":   "Style Grid Items",
					"default": true,
				},
			},
		},
		{
			Type:        "features",
			Name:        "Features",
			Description: "Showcase features with supporting images.",
			Category:    "marketing",
			Icon:        "sparkles",
			AllowedIn:   []string{"page", "post", "homepage"},
			Schema: map[string]interface{}{
				"title": map[string]interface{}{
					"type":  "string",
					"label": "Section Title",
				},
			},
		},
		{
			Type:        "posts_list",
			Name:        "Posts List",
			Description: "Display a list of blog posts",
			Category:    "content",
			Icon:        "list",
			AllowedIn:   []string{"page", "homepage"},
			Schema: map[string]interface{}{
				"title": map[string]interface{}{
					"type":  "string",
					"label": "Section Title",
				},
				"display_mode": map[string]interface{}{
					"type":  "select",
					"label": "Display mode",
					"options": []map[string]string{
						{"value": constants.PostListDisplayLimited, "label": "Limited (latest posts)"},
						{"value": constants.PostListDisplayPaginated, "label": "Paginated (all posts)"},
						{"value": constants.PostListDisplaySelected, "label": "Selected posts"},
					},
					"default": constants.PostListDisplayLimited,
				},
				"limit": map[string]interface{}{
					"type":         "number",
					"label":        "Number of Posts",
					"perPageLabel": "Number of posts to display on a page",
					"min":          1,
					"max":          constants.MaxPostListSectionLimit,
					"default":      constants.DefaultPostListSectionLimit,
				},
				"selected_posts": map[string]interface{}{
					"type":            "text",
					"label":           "Selected posts",
					"placeholder":     "Choose posts to feature",
					"allowPostPicker": true,
				},
			},
		},
		{
			Type:        "categories_list",
			Name:        "Categories List",
			Description: "Display blog categories",
			Category:    "navigation",
			Icon:        "tag",
			AllowedIn:   []string{"page", "homepage"},
			Schema: map[string]interface{}{
				"title": map[string]interface{}{
					"type":  "string",
					"label": "Section Title",
				},
				"limit": map[string]interface{}{
					"type":    "number",
					"label":   "Number of Categories",
					"min":     1,
					"max":     constants.MaxCategoryListSectionLimit,
					"default": constants.DefaultCategoryListSectionLimit,
				},
			},
		},
		{
			Type:        "courses_list",
			Name:        "Courses List",
			Description: "Display available courses",
			Category:    "content",
			Icon:        "book",
			AllowedIn:   []string{"page", "homepage"},
			Schema: map[string]interface{}{
				"title": map[string]interface{}{
					"type":  "string",
					"label": "Section Title",
				},
				"limit": map[string]interface{}{
					"type":         "number",
					"label":        "Number of Courses",
					"perPageLabel": "Number of courses to display on a page",
					"min":          1,
					"max":          constants.MaxCourseListSectionLimit,
					"default":      constants.DefaultCourseListSectionLimit,
				},
				"display_mode": map[string]interface{}{
					"type":  "select",
					"label": "Course list layout",
					"options": []map[string]string{
						{"value": constants.CourseListDisplayLimited, "label": "Limited (latest courses)"},
						{"value": constants.CourseListDisplayPaginated, "label": "Paginated (all courses)"},
						{"value": constants.CourseListDisplaySelected, "label": "Selected courses"},
					},
					"default": constants.CourseListDisplayLimited,
				},
				"selected_courses": map[string]interface{}{
					"type":              "text",
					"label":             "Selected courses",
					"placeholder":       "Choose courses to feature",
					"allowCoursePicker": true,
				},
				"show_all_button": map[string]interface{}{
					"type":    "boolean",
					"label":   "Show link to all courses",
					"default": false,
				},
				"all_courses_url": map[string]interface{}{
					"type":        "string",
					"label":       "All courses link",
					"placeholder": "/courses",
				},
				"all_courses_label": map[string]interface{}{
					"type":        "string",
					"label":       "All courses link label",
					"placeholder": "View all courses",
				},
			},
		},
		{
			Type:        "paragraph",
			Name:        "Paragraph",
			Description: "Text content",
			Category:    "elements",
			Icon:        "type",
			AllowedIn:   []string{"standard", "grid"},
			Schema: map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "textarea",
					"label":       "Text Content",
					"placeholder": "Enter text...",
					"required":    true,
				},
			},
		},
		{
			Type:        "image",
			Name:        "Image",
			Description: "Single image with caption",
			Category:    "media",
			Icon:        "image",
			AllowedIn:   []string{"standard", "grid"},
			Schema: map[string]interface{}{
				"url": map[string]interface{}{
					"type":     "image",
					"label":    "Image URL",
					"required": true,
				},
				"alt": map[string]interface{}{
					"type":        "string",
					"label":       "Alt Text",
					"placeholder": "Describe the image",
				},
				"caption": map[string]interface{}{
					"type":        "string",
					"label":       "Caption",
					"placeholder": "Optional caption",
				},
			},
		},
		{
			Type:        "image_group",
			Name:        "Image Gallery",
			Description: "Multiple images in a group",
			Category:    "media",
			Icon:        "images",
			AllowedIn:   []string{"standard", "grid"},
			Schema: map[string]interface{}{
				"images": map[string]interface{}{
					"type":     "array",
					"label":    "Images",
					"itemType": "image",
					"minItems": 1,
					"maxItems": 20,
				},
			},
		},
		{
			Type:        "list",
			Name:        "List",
			Description: "Bulleted or numbered list",
			Category:    "elements",
			Icon:        "list",
			AllowedIn:   []string{"standard", "grid"},
			Schema: map[string]interface{}{
				"items": map[string]interface{}{
					"type":     "array",
					"label":    "List Items",
					"itemType": "string",
					"minItems": 1,
				},
				"ordered": map[string]interface{}{
					"type":    "boolean",
					"label":   "Numbered List",
					"default": false,
				},
			},
		},
		{
			Type:        "search",
			Name:        "Search Box",
			Description: "Search functionality",
			Category:    "interactive",
			Icon:        "search",
			AllowedIn:   []string{"page", "homepage"},
		},
		{
			Type:        "feature_item",
			Name:        "Feature Item",
			Description: "Headline, supporting text, and optional image for the features section.",
			Category:    "elements",
			Icon:        "sparkles",
			AllowedIn:   []string{"features"},
			Schema: map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"label":       "Feature title",
					"placeholder": "Summarize the feature highlight",
					"required":    false,
				},
				"text": map[string]interface{}{
					"type":        "textarea",
					"label":       "Feature text",
					"placeholder": "Explain the feature value",
					"required":    true,
				},
				"image_url": map[string]interface{}{
					"type":     "image",
					"label":    "Image URL",
					"required": false,
				},
				"image_alt": map[string]interface{}{
					"type":        "string",
					"label":       "Image alt text",
					"placeholder": "Describe the image",
				},
			},
		},
	}
}
