package repository

import (
	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
)

type SearchRepository interface {
	SearchPosts(query string, limit int) ([]models.Post, error)
	SearchByTitle(query string, limit int) ([]models.Post, error)
	SearchByContent(query string, limit int) ([]models.Post, error)
	SearchByTag(tag string, limit int) ([]models.Post, error)
	SearchByAuthor(author string, limit int) ([]models.Post, error)
}

type searchRepository struct {
	db *gorm.DB
}

func NewSearchRepository(db *gorm.DB) SearchRepository {
	db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_posts_title_content_tsvector 
		ON posts USING GIN (to_tsvector('english', title || ' ' || content))
	`)

	return &searchRepository{db: db}
}

func (r *searchRepository) SearchPosts(query string, limit int) ([]models.Post, error) {
	var posts []models.Post

	err := r.db.Where(
		"to_tsvector('english', title || ' ' || content) @@ plainto_tsquery('english', ?)",
		query,
	).
		Where("published = ?", true).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("posts.created_at DESC").
		Limit(limit).
		Find(&posts).Error

	return posts, err
}

func (r *searchRepository) SearchByTitle(query string, limit int) ([]models.Post, error) {
	var posts []models.Post

	err := r.db.Where("title ILIKE ?", "%"+query+"%").
		Where("published = ?", true).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("posts.created_at DESC").
		Limit(limit).
		Find(&posts).Error

	return posts, err
}

func (r *searchRepository) SearchByContent(query string, limit int) ([]models.Post, error) {
	var posts []models.Post

	err := r.db.Where("content ILIKE ?", "%"+query+"%").
		Where("published = ?", true).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("posts.created_at DESC").
		Limit(limit).
		Find(&posts).Error

	return posts, err
}

func (r *searchRepository) SearchByTag(tag string, limit int) ([]models.Post, error) {
	var posts []models.Post

	err := r.db.Joins("JOIN post_tags ON post_tags.post_id = posts.id").
		Joins("JOIN tags ON tags.id = post_tags.tag_id").
		Where("tags.slug = ? OR tags.name ILIKE ?", tag, "%"+tag+"%").
		Where("posts.published = ?", true).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Group("posts.id").
		Order("posts.created_at DESC").
		Limit(limit).
		Find(&posts).Error

	return posts, err
}

func (r *searchRepository) SearchByAuthor(author string, limit int) ([]models.Post, error) {
	var posts []models.Post

	err := r.db.Joins("JOIN users ON users.id = posts.author_id").
		Where("users.username ILIKE ?", "%"+author+"%").
		Where("posts.published = ?", true).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("posts.created_at DESC").
		Limit(limit).
		Find(&posts).Error

	return posts, err
}
