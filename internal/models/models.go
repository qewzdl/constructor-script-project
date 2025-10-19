package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"strings"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Username string `gorm:"uniqueIndex;not null" json:"username"`
	Email    string `gorm:"uniqueIndex;not null" json:"email"`
	Password string `gorm:"not null" json:"-"`
	Role     string `gorm:"default:'user'" json:"role"`

	Status string `gorm:"default:'active'" json:"status"`

	Posts    []Post    `gorm:"foreignKey:AuthorID" json:"posts,omitempty"`
	Comments []Comment `gorm:"foreignKey:AuthorID" json:"comments,omitempty"`
}

type Category struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string `gorm:"uniqueIndex;not null" json:"name"`
	Slug        string `gorm:"uniqueIndex;not null" json:"slug"`
	Description string `json:"description"`

	Order int `gorm:"default:0" json:"order"`

	Posts []Post `gorm:"foreignKey:CategoryID" json:"posts,omitempty"`
}

type Post struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title       string `gorm:"not null" json:"title"`
	Slug        string `gorm:"uniqueIndex;not null" json:"slug"`
	Description string `json:"description"`
	Content     string `gorm:"type:text;not null" json:"content"`
	Excerpt     string `json:"excerpt"`
	FeaturedImg string `json:"featured_img"`
	Published   bool   `gorm:"default:false" json:"published"`
	Views       int    `gorm:"default:0" json:"views"`

	Sections PostSections `gorm:"type:jsonb" json:"sections"`
	Template string       `gorm:"default:'post'" json:"template"`

	AuthorID   uint     `gorm:"not null" json:"author_id"`
	Author     User     `gorm:"foreignKey:AuthorID" json:"author"`
	CategoryID uint     `json:"category_id"`
	Category   Category `gorm:"foreignKey:CategoryID" json:"category"`

	Tags     []Tag     `gorm:"many2many:post_tags;" json:"tags,omitempty"`
	Comments []Comment `gorm:"foreignKey:PostID" json:"comments,omitempty"`
}

type Tag struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string     `gorm:"uniqueIndex;not null" json:"name"`
	Slug        string     `gorm:"uniqueIndex;not null" json:"slug"`
	UnusedSince *time.Time `gorm:"index" json:"unused_since,omitempty"`
	Posts       []Post     `gorm:"many2many:post_tags;" json:"posts,omitempty"`
}

type Comment struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Content  string `gorm:"type:text;not null" json:"content"`
	Approved bool   `gorm:"default:true" json:"approved"`

	PostID uint `gorm:"not null" json:"post_id"`
	Post   Post `gorm:"foreignKey:PostID" json:"post,omitempty"`

	AuthorID uint `gorm:"not null" json:"author_id"`
	Author   User `gorm:"foreignKey:AuthorID" json:"author"`

	ParentID *uint      `json:"parent_id"`
	Parent   *Comment   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Replies  []*Comment `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type CreateCommentRequest struct {
	Content  string `json:"content" binding:"required"`
	ParentID *uint  `json:"parent_id"`
}

type UpdateCommentRequest struct {
	Content  string `json:"content" binding:"required"`
	Approved *bool  `json:"approved"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type PostSections []Section

type Section struct {
	ID       string           `json:"id"`
	Title    string           `json:"title"`
	Image    string           `json:"image"`
	Order    int              `json:"order"`
	Elements []SectionElement `json:"elements"`
}

type SectionElement struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Order   int         `json:"order"`
	Content interface{} `json:"content"`
}

type ParagraphContent struct {
	Text string `json:"text"`
}

type ImageContent struct {
	URL     string `json:"url"`
	Alt     string `json:"alt"`
	Caption string `json:"caption"`
}

type ImageGroupContent struct {
	Images []ImageContent `json:"images"`
	Layout string         `json:"layout"`
}

type ListContent struct {
	Items   []string `json:"items"`
	Ordered bool     `json:"ordered"`
}

func (ps *PostSections) Scan(value interface{}) error {
	if value == nil {
		*ps = PostSections{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan PostSections")
	}

	return json.Unmarshal(bytes, ps)
}

func (ps PostSections) Value() (driver.Value, error) {
	if len(ps) == 0 {
		return nil, nil
	}
	return json.Marshal(ps)
}

type CreatePostRequest struct {
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Excerpt     string    `json:"excerpt"`
	FeaturedImg string    `json:"featured_img"`
	Published   bool      `json:"published"`
	CategoryID  uint      `json:"category_id"`
	TagNames    []string  `json:"tags"`
	Sections    []Section `json:"sections"`
	Template    string    `json:"template"`
}

type UpdatePostRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Content     *string    `json:"content"`
	Excerpt     *string    `json:"excerpt"`
	FeaturedImg *string    `json:"featured_img"`
	Published   *bool      `json:"published"`
	CategoryID  *uint      `json:"category_id"`
	TagNames    []string   `json:"tags"`
	Sections    *[]Section `json:"sections"`
	Template    *string    `json:"template"`
}

type Page struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title       string       `gorm:"not null" json:"title"`
	Slug        string       `gorm:"uniqueIndex;not null" json:"slug"`
	Description string       `json:"description"`
	FeaturedImg string       `json:"featured_img"`
	Published   bool         `gorm:"default:false" json:"published"`
	Sections    PostSections `gorm:"type:jsonb" json:"sections"`
	Template    string       `gorm:"default:'page'" json:"template"`

	Order int `gorm:"default:0" json:"order"`
}

type CreatePageRequest struct {
	Title       string    `json:"title" binding:"required"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	FeaturedImg string    `json:"featured_img"`
	Published   bool      `json:"published"`
	Sections    []Section `json:"sections"`
	Template    string    `json:"template"`
	Order       int       `json:"order"`
}

type UpdatePageRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	FeaturedImg *string    `json:"featured_img"`
	Published   *bool      `json:"published"`
	Sections    *[]Section `json:"sections"`
	Template    *string    `json:"template"`
	Order       *int       `json:"order"`
}

type Setting struct {
	Key       string    `gorm:"primaryKey;size:191" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SiteSettings struct {
	Name                    string       `json:"name"`
	Description             string       `json:"description"`
	URL                     string       `json:"url"`
	Favicon                 string       `json:"favicon"`
	FaviconType             string       `json:"favicon_type"`
	Logo                    string       `json:"logo"`
	UnusedTagRetentionHours int          `json:"unused_tag_retention_hours"`
	SocialLinks             []SocialLink `json:"social_links"`
}

type UpdateSiteSettingsRequest struct {
	Name                    string `json:"name" binding:"required"`
	Description             string `json:"description"`
	URL                     string `json:"url" binding:"required"`
	Favicon                 string `json:"favicon"`
	Logo                    string `json:"logo"`
	UnusedTagRetentionHours int    `json:"unused_tag_retention_hours" binding:"required,min=1"`
}

func DetectFaviconType(favicon string) string {
	const defaultType = "image/x-icon"

	trimmed := strings.TrimSpace(favicon)
	if trimmed == "" {
		return defaultType
	}

	value := trimmed
	if parsed, err := url.Parse(trimmed); err == nil {
		if parsed.Path != "" {
			value = parsed.Path
		}
	}

	ext := strings.TrimPrefix(strings.ToLower(path.Ext(value)), ".")
	switch ext {
	case "png":
		return "image/png"
	case "svg":
		return "image/svg+xml"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "ico":
		return "image/x-icon"
	default:
		return defaultType
	}
}

type SetupRequest struct {
	AdminUsername string `json:"admin_username" binding:"required,min=3,max=50"`
	AdminEmail    string `json:"admin_email" binding:"required,email"`
	AdminPassword string `json:"admin_password" binding:"required,min=8"`

	SiteName        string `json:"site_name" binding:"required"`
	SiteDescription string `json:"site_description"`
	SiteURL         string `json:"site_url" binding:"required"`
	SiteFavicon     string `json:"site_favicon"`
	SiteLogo        string `json:"site_logo"`
}

type SocialLink struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name  string `gorm:"not null" json:"name"`
	URL   string `gorm:"not null" json:"url"`
	Icon  string `json:"icon"`
	Order int    `gorm:"default:0" json:"order"`
}

type CreateSocialLinkRequest struct {
	Name  string `json:"name" binding:"required"`
	URL   string `json:"url" binding:"required"`
	Icon  string `json:"icon"`
	Order *int   `json:"order"`
}

type UpdateSocialLinkRequest struct {
	Name  string `json:"name" binding:"required"`
	URL   string `json:"url" binding:"required"`
	Icon  string `json:"icon"`
	Order *int   `json:"order"`
}
