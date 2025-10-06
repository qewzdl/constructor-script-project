package models

import (
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
	Role     string `gorm:"default:'user'" json:"role"` // user, admin

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
	Content     string `gorm:"type:text;not null" json:"content"`
	Excerpt     string `json:"excerpt"`
	FeaturedImg string `json:"featured_img"`
	Published   bool   `gorm:"default:false" json:"published"`
	Views       int    `gorm:"default:0" json:"views"`

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

	Name  string `gorm:"uniqueIndex;not null" json:"name"`
	Slug  string `gorm:"uniqueIndex;not null" json:"slug"`
	Posts []Post `gorm:"many2many:post_tags;" json:"posts,omitempty"`
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

// DTO for requests

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CreatePostRequest struct {
	Title       string   `json:"title" binding:"required"`
	Content     string   `json:"content" binding:"required"`
	Excerpt     string   `json:"excerpt"`
	FeaturedImg string   `json:"featured_img"`
	Published   bool     `json:"published"`
	CategoryID  uint     `json:"category_id"`
	TagNames    []string `json:"tags"`
}

type UpdatePostRequest struct {
	Title       *string  `json:"title"`
	Content     *string  `json:"content"`
	Excerpt     *string  `json:"excerpt"`
	FeaturedImg *string  `json:"featured_img"`
	Published   *bool    `json:"published"`
	CategoryID  *uint    `json:"category_id"`
	TagNames    []string `json:"tags"`
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
