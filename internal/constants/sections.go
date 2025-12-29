package constants

import "strings"

const (
	// DefaultPostListSectionLimit defines the default number of posts shown in a post list section.
	DefaultPostListSectionLimit = 6
	// MaxPostListSectionLimit defines an upper bound to avoid rendering overly large post lists.
	MaxPostListSectionLimit = 24

	// DefaultCategoryListSectionLimit defines the default number of categories shown in a category list section.
	DefaultCategoryListSectionLimit = 10
	// MaxCategoryListSectionLimit defines an upper bound for category list sections to keep navigation manageable.
	MaxCategoryListSectionLimit = 30

	// PostListDisplayLimited shows a capped number of the most recent posts.
	PostListDisplayLimited = "limited"
	// PostListDisplayPaginated shows all posts with pagination.
	PostListDisplayPaginated = "paginated"
	// PostListDisplaySelected shows only administrator-selected posts (with optional pagination).
	PostListDisplaySelected = "selected"
	// PostListDisplayCarousel shows posts inside a horizontal carousel.
	PostListDisplayCarousel = "carousel"

	// DefaultCourseListSectionLimit defines the default number of courses shown in a course list section.
	DefaultCourseListSectionLimit = 3
	// MaxCourseListSectionLimit defines an upper bound for course list sections to keep layouts balanced.
	MaxCourseListSectionLimit = 12

	// CourseListModeCatalog renders publicly available course packages that can be purchased or enrolled in.
	CourseListModeCatalog = "catalog"
	// CourseListModeOwned renders the courses currently granted to the authenticated user.
	CourseListModeOwned = "owned"

	// CourseListDisplayLimited shows a capped number of the most recent courses.
	CourseListDisplayLimited = "limited"
	// CourseListDisplayPaginated shows all courses with pagination.
	CourseListDisplayPaginated = "paginated"
	// CourseListDisplaySelected shows only administrator-selected courses (with optional pagination).
	CourseListDisplaySelected = "selected"
	// CourseListDisplayCarousel shows courses inside a horizontal carousel.
	CourseListDisplayCarousel = "carousel"

	// DefaultCarouselColumns defines how many items are shown in a carousel by default.
	DefaultCarouselColumns = 3
	// MinCarouselColumns sets the lower bound for carousel column count.
	MinCarouselColumns = 1
	// MaxCarouselColumns sets the upper bound for carousel column count.
	MaxCarouselColumns = 4

	// DefaultSectionPadding defines the default vertical padding (in pixels) applied to newly created sections.
	DefaultSectionPadding = 64
	// DefaultSectionMargin defines the default vertical margin (in pixels) applied to newly created sections.
	DefaultSectionMargin = 0

	// DefaultSectionAnimation defines the default scroll animation applied to sections.
	DefaultSectionAnimation = "float-up"
	// DefaultSectionAnimationBlur controls whether blur is applied during the section animation.
	DefaultSectionAnimationBlur = true
)

var sectionPaddingOptions = []int{0, 4, 8, 16, 32, 64, 128}
var sectionMarginOptions = []int{0, 4, 8, 16, 32, 64, 128}
var sectionAnimationOptions = []SectionAnimationOption{
	{
		Value:       "float-up",
		Label:       "Float up",
		Description: "Tilted lift with a soft blur fade",
	},
	{
		Value:       "fade-in",
		Label:       "Fade in",
		Description: "Gentle fade with a slight rise",
	},
	{
		Value:       "slide-left",
		Label:       "Slide from right",
		Description: "Horizontal slide-in with easing",
	},
	{
		Value:       "zoom-in",
		Label:       "Zoom in",
		Description: "Scale up softly from the center",
	},
	{
		Value:       "none",
		Label:       "None",
		Description: "Disable section animation",
	},
}

// SectionAnimationOption describes an available section animation.
type SectionAnimationOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// SectionPaddingOptions returns the allowed vertical padding options for sections in pixels.
// A copy of the slice is returned to prevent external mutation of the internal list.
func SectionPaddingOptions() []int {
	options := make([]int, len(sectionPaddingOptions))
	copy(options, sectionPaddingOptions)
	return options
}

// SectionMarginOptions returns the allowed vertical margin options for sections in pixels.
// A copy of the slice is returned to prevent external mutation of the internal list.
func SectionMarginOptions() []int {
	options := make([]int, len(sectionMarginOptions))
	copy(options, sectionMarginOptions)
	return options
}

// SectionAnimationOptions returns the allowed section animations.
// A copy of the slice is returned to prevent external mutation of the internal list.
func SectionAnimationOptions() []SectionAnimationOption {
	options := make([]SectionAnimationOption, len(sectionAnimationOptions))
	copy(options, sectionAnimationOptions)
	return options
}

// NormaliseSectionAnimation returns a known animation value or the default.
func NormaliseSectionAnimation(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return DefaultSectionAnimation
	}
	for _, option := range sectionAnimationOptions {
		if option.Value == trimmed {
			return trimmed
		}
	}
	return DefaultSectionAnimation
}

// NormaliseSectionAnimationBlur returns whether blur should be applied for the animation.
func NormaliseSectionAnimationBlur(value *bool) bool {
	if value == nil {
		return DefaultSectionAnimationBlur
	}
	return *value
}
