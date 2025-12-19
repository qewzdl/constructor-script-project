package constants

const (
	// DefaultPostListSectionLimit defines the default number of posts shown in a post list section.
	DefaultPostListSectionLimit = 6
	// MaxPostListSectionLimit defines an upper bound to avoid rendering overly large post lists.
	MaxPostListSectionLimit = 24

	// DefaultCategoryListSectionLimit defines the default number of categories shown in a category list section.
	DefaultCategoryListSectionLimit = 10
	// MaxCategoryListSectionLimit defines an upper bound for category list sections to keep navigation manageable.
	MaxCategoryListSectionLimit = 30

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

	// DefaultSectionPadding defines the default vertical padding (in pixels) applied to newly created sections.
	DefaultSectionPadding = 64
	// DefaultSectionMargin defines the default vertical margin (in pixels) applied to newly created sections.
	DefaultSectionMargin = 0
)

var sectionPaddingOptions = []int{0, 4, 8, 16, 32, 64, 128}
var sectionMarginOptions = []int{0, 4, 8, 16, 32, 64, 128}

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
