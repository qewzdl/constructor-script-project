package repository

import (
	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
)

type CommentRepository interface {
	Create(comment *models.Comment) error
	GetByID(id uint) (*models.Comment, error)
	GetByPostID(postID uint) ([]models.Comment, error)
	GetAll() ([]models.Comment, error)
	Update(comment *models.Comment) error
	Delete(id uint) error
}

type commentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) Create(comment *models.Comment) error {
	return r.db.Create(comment).Error
}

func (r *commentRepository) GetByID(id uint) (*models.Comment, error) {
	var comment models.Comment
	err := r.db.Preload("Author").Preload("Replies.Author").First(&comment, id).Error
	return &comment, err
}

func (r *commentRepository) GetByPostID(postID uint) ([]models.Comment, error) {
	var comments []models.Comment
	err := r.db.Where("post_id = ? AND parent_id IS NULL", postID).
		Preload("Author").
		Preload("Replies.Author").
		Preload("Replies.Replies.Author").
		Order("created_at DESC").
		Find(&comments).Error
	return comments, err
}

func (r *commentRepository) GetAll() ([]models.Comment, error) {
	var comments []models.Comment
	err := r.db.Preload("Author").Preload("Post").Order("created_at DESC").Find(&comments).Error
	return comments, err
}

func (r *commentRepository) Update(comment *models.Comment) error {
	return r.db.Save(comment).Error
}

func (r *commentRepository) Delete(id uint) error {
	return r.db.Delete(&models.Comment{}, id).Error
}

func (r *commentRepository) GetPending() ([]models.Comment, error) {
	var comments []models.Comment
	err := r.db.Where("approved = ?", false).
		Preload("Author").
		Preload("Post").
		Order("created_at DESC").
		Find(&comments).Error
	return comments, err
}

func (r *commentRepository) GetByUserID(userID uint) ([]models.Comment, error) {
	var comments []models.Comment
	err := r.db.Where("author_id = ?", userID).
		Preload("Post").
		Order("created_at DESC").
		Find(&comments).Error
	return comments, err
}

func (r *commentRepository) CountByPostID(postID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Comment{}).
		Where("post_id = ?", postID).
		Count(&count).Error
	return count, err
}
