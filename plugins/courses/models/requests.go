package models

// CreatePackageRequest represents a request to create a course package
type CreatePackageRequest struct {
	Title           string `json:"title" binding:"required"`
	Slug            string `json:"slug"`
	Summary         string `json:"summary"`
	Description     string `json:"description"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	PriceCents      int64  `json:"price_cents"`
	ImageURL        string `json:"image_url"`
}

// UpdatePackageRequest represents a request to update a course package
type UpdatePackageRequest struct {
	Title           *string `json:"title"`
	Slug            *string `json:"slug"`
	Summary         *string `json:"summary"`
	Description     *string `json:"description"`
	MetaTitle       *string `json:"meta_title"`
	MetaDescription *string `json:"meta_description"`
	PriceCents      *int64  `json:"price_cents"`
	ImageURL        *string `json:"image_url"`
}

// CreateTopicRequest represents a request to create a topic
type CreateTopicRequest struct {
	Title           string `json:"title" binding:"required"`
	Slug            string `json:"slug"`
	Summary         string `json:"summary"`
	Description     string `json:"description"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
}

// UpdateTopicRequest represents a request to update a topic
type UpdateTopicRequest struct {
	Title           *string `json:"title"`
	Slug            *string `json:"slug"`
	Summary         *string `json:"summary"`
	Description     *string `json:"description"`
	MetaTitle       *string `json:"meta_title"`
	MetaDescription *string `json:"meta_description"`
}

// CreateVideoRequest represents a request to create a video
type CreateVideoRequest struct {
	Title           string           `json:"title" binding:"required"`
	Description     string           `json:"description"`
	FileURL         string           `json:"file_url" binding:"required"`
	Filename        string           `json:"filename" binding:"required"`
	DurationSeconds int              `json:"duration_seconds" binding:"required"`
	Sections        Sections         `json:"sections"`
	Attachments     VideoAttachments `json:"attachments"`
}

// UpdateVideoRequest represents a request to update a video
type UpdateVideoRequest struct {
	Title           *string          `json:"title"`
	Description     *string          `json:"description"`
	FileURL         *string          `json:"file_url"`
	Filename        *string          `json:"filename"`
	DurationSeconds *int             `json:"duration_seconds"`
	Sections        Sections         `json:"sections"`
	Attachments     VideoAttachments `json:"attachments"`
}

// CreateTestRequest represents a request to create a test
type CreateTestRequest struct {
	Title       string                      `json:"title" binding:"required"`
	Description string                      `json:"description"`
	Questions   []CreateTestQuestionRequest `json:"questions"`
}

// CreateTestQuestionRequest represents a request to create a test question
type CreateTestQuestionRequest struct {
	Prompt      string                            `json:"prompt" binding:"required"`
	Type        string                            `json:"type" binding:"required"`
	Explanation string                            `json:"explanation"`
	AnswerText  string                            `json:"answer_text"`
	Position    int                               `json:"position"`
	Options     []CreateTestQuestionOptionRequest `json:"options"`
}

// CreateTestQuestionOptionRequest represents a request to create a question option
type CreateTestQuestionOptionRequest struct {
	Text     string `json:"text" binding:"required"`
	Correct  bool   `json:"correct"`
	Position int    `json:"position"`
}

// GrantAccessRequest represents a request to grant package access
type GrantAccessRequest struct {
	UserID    uint   `json:"user_id" binding:"required"`
	PackageID uint   `json:"package_id" binding:"required"`
	ExpiresAt *int64 `json:"expires_at"` // Unix timestamp
}

// CheckoutRequest represents a request to checkout a package
type CheckoutRequest struct {
	PackageID     uint   `json:"package_id" binding:"required,gt=0"`
	CustomerEmail string `json:"customer_email" binding:"omitempty,email"`
}

// CheckoutSession represents a checkout session
type CheckoutSession struct {
	SessionID   string `json:"session_id"`
	CheckoutURL string `json:"checkout_url"`
}

// SubmitTestRequest represents a request to submit test answers
type SubmitTestRequest struct {
	TestID  uint                 `json:"test_id" binding:"required"`
	Answers map[uint]interface{} `json:"answers" binding:"required"` // question_id -> answer
}

// TestResultResponse represents test result
type TestResultResponse struct {
	Score      int  `json:"score"`
	MaxScore   int  `json:"max_score"`
	Percentage int  `json:"percentage"`
	Passed     bool `json:"passed"`
}
