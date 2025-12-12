package models

import "time"

// CreatePostRequest represents request to create a new post
type CreatePostRequest struct {
	Title       string       `json:"title" binding:"required"`
	Slug        string       `json:"slug"`
	Description string       `json:"description"`
	Content     string       `json:"content" binding:"required"`
	Excerpt     string       `json:"excerpt"`
	FeaturedImg string       `json:"featured_img"`
	Published   bool         `json:"published"`
	PublishAt   *time.Time   `json:"publish_at"`
	CategoryID  uint         `json:"category_id"`
	TagIDs      []uint       `json:"tag_ids"`
	Sections    PostSections `json:"sections"`
	Template    string       `json:"template"`
}

// UpdatePostRequest represents request to update a post
type UpdatePostRequest struct {
	Title       *string      `json:"title"`
	Slug        *string      `json:"slug"`
	Description *string      `json:"description"`
	Content     *string      `json:"content"`
	Excerpt     *string      `json:"excerpt"`
	FeaturedImg *string      `json:"featured_img"`
	Published   *bool        `json:"published"`
	PublishAt   *time.Time   `json:"publish_at"`
	CategoryID  *uint        `json:"category_id"`
	TagIDs      []uint       `json:"tag_ids"`
	Sections    PostSections `json:"sections"`
	Template    *string      `json:"template"`
}

// CreateCategoryRequest represents request to create a category
type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Order       int    `json:"order"`
}

// UpdateCategoryRequest represents request to update a category
type UpdateCategoryRequest struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
	Order       *int    `json:"order"`
}

// CreateCommentRequest represents request to create a comment
type CreateCommentRequest struct {
	Content  string `json:"content" binding:"required"`
	PostID   uint   `json:"post_id" binding:"required"`
	ParentID *uint  `json:"parent_id"`
}

// UpdateCommentRequest represents request to update a comment
type UpdateCommentRequest struct {
	Content  *string `json:"content"`
	Approved *bool   `json:"approved"`
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query      string `form:"q" json:"query" binding:"required"`
	Page       int    `form:"page" json:"page"`
	PageSize   int    `form:"page_size" json:"page_size"`
	CategoryID *uint  `form:"category_id" json:"category_id"`
	TagID      *uint  `form:"tag_id" json:"tag_id"`
}
