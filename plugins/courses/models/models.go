package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	StepTypeVideo   = "video"
	StepTypeTest    = "test"
	StepTypeContent = "content"
)

const (
	TestQuestionTypeText           = "text"
	TestQuestionTypeSingleChoice   = "single_choice"
	TestQuestionTypeMultipleChoice = "multiple_choice"
)

// CoursePackage represents a course package
type CoursePackage struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title              string `gorm:"not null" json:"title"`
	Slug               string `gorm:"not null;uniqueIndex:idx_course_packages_slug,where:deleted_at IS NULL" json:"slug"`
	Summary            string `json:"summary"`
	Description        string `json:"description"`
	MetaTitle          string `json:"meta_title"`
	MetaDescription    string `json:"meta_description"`
	PriceCents         int64  `gorm:"not null" json:"price_cents"`
	DiscountPriceCents *int64 `json:"discount_price_cents,omitempty"`
	ImageURL           string `json:"image_url"`

	Topics []CourseTopic `gorm:"-" json:"topics"`
}

// CourseTopic represents a course topic
type CourseTopic struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title           string `gorm:"not null" json:"title"`
	Slug            string `gorm:"not null;uniqueIndex:idx_course_topics_slug,where:deleted_at IS NULL" json:"slug"`
	Summary         string `json:"summary"`
	Description     string `json:"description"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`

	Steps []CourseTopicStep `gorm:"-" json:"steps"`
}

// CourseVideo represents a video in a course
type CourseVideo struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title           string `gorm:"not null" json:"title"`
	Description     string `json:"description"`
	FileURL         string `gorm:"not null" json:"file_url"`
	Filename        string `gorm:"not null" json:"filename"`
	DurationSeconds int    `gorm:"not null" json:"duration_seconds"`

	Sections     Sections         `gorm:"type:jsonb" json:"sections"`
	Attachments  VideoAttachments `gorm:"type:jsonb" json:"attachments"`
	SectionsHTML string           `gorm:"-" json:"sections_html,omitempty"`
}

// CourseContent represents text content in a course
type CourseContent struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title        string   `gorm:"not null" json:"title"`
	Description  string   `json:"description"`
	Sections     Sections `gorm:"type:jsonb" json:"sections"`
	SectionsHTML string   `gorm:"-" json:"sections_html,omitempty"`
}

// CourseTest represents a test in a course
type CourseTest struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title       string `gorm:"not null" json:"title"`
	Description string `json:"description"`

	Questions []CourseTestQuestion `gorm:"-" json:"questions"`
}

// CourseTestQuestion represents a question in a test
type CourseTestQuestion struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	TestID      uint   `gorm:"not null;index" json:"test_id"`
	Prompt      string `gorm:"not null" json:"prompt"`
	Type        string `gorm:"type:varchar(32);not null" json:"type"`
	Explanation string `json:"explanation"`
	AnswerText  string `json:"answer_text"`
	Position    int    `gorm:"not null;default:0" json:"position"`

	Options []CourseTestQuestionOption `gorm:"-" json:"options"`
}

// CourseTestQuestionOption represents an option for a test question
type CourseTestQuestionOption struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	QuestionID uint   `gorm:"not null;index" json:"question_id"`
	Text       string `gorm:"not null" json:"text"`
	Correct    bool   `gorm:"not null" json:"correct"`
	Position   int    `gorm:"not null;default:0" json:"position"`
}

// CourseTopicStep represents a step in a course topic
type CourseTopicStep struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	TopicID  uint   `gorm:"not null;index" json:"topic_id"`
	StepType string `gorm:"type:varchar(32);not null;index" json:"type"`
	Position int    `gorm:"not null;default:0" json:"position"`

	VideoID   *uint `gorm:"index" json:"video_id,omitempty"`
	TestID    *uint `gorm:"index" json:"test_id,omitempty"`
	ContentID *uint `gorm:"index" json:"content_id,omitempty"`

	Video   *CourseVideo   `gorm:"-" json:"video,omitempty"`
	Test    *CourseTest    `gorm:"-" json:"test,omitempty"`
	Content *CourseContent `gorm:"-" json:"content,omitempty"`
}

// CoursePackageTopic links packages and topics
type CoursePackageTopic struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	PackageID uint `gorm:"not null;index" json:"package_id"`
	TopicID   uint `gorm:"not null;index" json:"topic_id"`
	Position  int  `gorm:"not null;default:0" json:"position"`

	Package CoursePackage `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
	Topic   CourseTopic   `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

// CoursePackageAccess tracks user access to packages
type CoursePackageAccess struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID    uint `gorm:"not null;index;uniqueIndex:idx_course_package_access_user_package,priority:1" json:"user_id"`
	PackageID uint `gorm:"not null;index;uniqueIndex:idx_course_package_access_user_package,priority:2" json:"package_id"`

	GrantedBy *uint      `gorm:"index" json:"granted_by,omitempty"`
	ExpiresAt *time.Time `gorm:"index" json:"expires_at,omitempty"`
}

// CourseTestResult tracks test results
type CourseTestResult struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	TestID uint `gorm:"not null;index" json:"test_id"`
	UserID uint `gorm:"not null;index" json:"user_id"`

	Score    int    `gorm:"not null" json:"score"`
	MaxScore int    `gorm:"not null" json:"max_score"`
	Answers  []byte `gorm:"type:jsonb" json:"answers"`
}

// Custom types for JSON fields
type Sections []Section

type Section struct {
	Type    string                 `json:"type"`
	Content map[string]interface{} `json:"content"`
}

type VideoAttachments []VideoAttachment

type VideoAttachment struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Size int64  `json:"size"`
}

// Helper types
type UserCoursePackage struct {
	Package CoursePackage       `json:"package"`
	Access  CoursePackageAccess `json:"access"`
}
