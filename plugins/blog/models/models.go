package models

import (
	"time"

	"gorm.io/gorm"
)

// Category represents a blog post category
type Category struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string `gorm:"uniqueIndex;not null" json:"name"`
	Slug        string `gorm:"uniqueIndex;not null" json:"slug"`
	Description string `json:"description"`
	Order       int    `gorm:"default:0" json:"order"`

	Posts []Post `gorm:"foreignKey:CategoryID" json:"posts,omitempty"`
}

// Post represents a blog post
type Post struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title       string     `gorm:"not null" json:"title"`
	Slug        string     `gorm:"uniqueIndex;not null" json:"slug"`
	Description string     `json:"description"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	Excerpt     string     `json:"excerpt"`
	FeaturedImg string     `json:"featured_img"`
	Published   bool       `gorm:"default:false" json:"published"`
	PublishAt   *time.Time `gorm:"index" json:"publish_at,omitempty"`
	PublishedAt *time.Time `gorm:"index" json:"published_at,omitempty"`
	Views       int        `gorm:"default:0" json:"views"`

	Sections PostSections `gorm:"type:jsonb" json:"sections"`
	Template string       `gorm:"default:'post'" json:"template"`

	AuthorID   uint     `gorm:"not null" json:"author_id"`
	CategoryID uint     `json:"category_id"`
	Category   Category `gorm:"foreignKey:CategoryID" json:"category"`

	Tags     []Tag     `gorm:"many2many:post_tags;" json:"tags,omitempty"`
	Comments []Comment `gorm:"foreignKey:PostID" json:"comments,omitempty"`
}

// Tag represents a post tag
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

// Comment represents a comment on a post
type Comment struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Content  string `gorm:"type:text;not null" json:"content"`
	Approved bool   `gorm:"default:true" json:"approved"`

	PostID   uint `gorm:"not null" json:"post_id"`
	Post     Post `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE" json:"post,omitempty"`
	AuthorID uint `gorm:"not null" json:"author_id"`

	ParentID *uint      `json:"parent_id"`
	Parent   *Comment   `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE" json:"parent,omitempty"`
	Replies  []*Comment `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE" json:"replies,omitempty"`
}

// PostViewStat tracks daily view statistics
type PostViewStat struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	PostID uint      `gorm:"not null;index:idx_post_view_stats_post_date,priority:1" json:"post_id"`
	Date   time.Time `gorm:"type:date;not null;index:idx_post_view_stats_post_date,priority:2" json:"date"`
	Views  int64     `gorm:"not null;default:0" json:"views"`

	Post Post `gorm:"foreignKey:PostID" json:"-"`
}

// PostSections holds the post content sections
type PostSections []PostSection

// PostSection represents a section in the post
type PostSection struct {
	Type    string                 `json:"type"`
	Content map[string]interface{} `json:"content"`
}
