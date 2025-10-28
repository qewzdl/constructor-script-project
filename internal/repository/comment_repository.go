package repository

import (
	"time"

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
	GetPending() ([]models.Comment, error)
	GetByUserID(userID uint) ([]models.Comment, error)
	CountByPostID(postID uint) (int64, error)
	DailyCountsByPostID(postID uint, start time.Time) ([]DailyCount, error)
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
	err := r.db.Where("post_id = ? AND parent_id IS NULL AND approved = ?", postID, true).
		Preload("Author").
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Where("approved = ?", true).Order("comments.created_at ASC")
		}).
		Preload("Replies.Author").
		Preload("Replies.Replies", func(db *gorm.DB) *gorm.DB {
			return db.Where("approved = ?", true).Order("comments.created_at ASC")
		}).
		Preload("Replies.Replies.Author").
		Order("comments.created_at ASC").
		Find(&comments).Error
	return comments, err
}

func (r *commentRepository) GetAll() ([]models.Comment, error) {
	var comments []models.Comment
	err := r.db.Preload("Author").Preload("Post").Order("comments.created_at DESC").Find(&comments).Error
	return comments, err
}

func (r *commentRepository) Update(comment *models.Comment) error {
	return r.db.Save(comment).Error
}

func (r *commentRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := r.deleteReplies(tx, id); err != nil {
			return err
		}

		return tx.Unscoped().Delete(&models.Comment{}, id).Error
	})
}

func (r *commentRepository) deleteReplies(tx *gorm.DB, parentID uint) error {
	var replyIDs []uint
	if err := tx.Model(&models.Comment{}).Where("parent_id = ?", parentID).Pluck("id", &replyIDs).Error; err != nil {
		return err
	}

	for _, replyID := range replyIDs {
		if err := r.deleteReplies(tx, replyID); err != nil {
			return err
		}

		if err := tx.Unscoped().Delete(&models.Comment{}, replyID).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *commentRepository) GetPending() ([]models.Comment, error) {
	var comments []models.Comment
	err := r.db.Where("approved = ?", false).
		Preload("Author").
		Preload("Post").
		Order("comments.created_at DESC").
		Find(&comments).Error
	return comments, err
}

func (r *commentRepository) GetByUserID(userID uint) ([]models.Comment, error) {
	var comments []models.Comment
	err := r.db.Where("author_id = ?", userID).
		Preload("Post").
		Order("comments.created_at DESC").
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

func (r *commentRepository) DailyCountsByPostID(postID uint, start time.Time) ([]DailyCount, error) {
	var counts []DailyCount

	query := r.db.Model(&models.Comment{}).
		Select("DATE_TRUNC('day', created_at) AS period, COUNT(*) AS count").
		Where("post_id = ?", postID)

	if !start.IsZero() {
		query = query.Where("created_at >= ?", start)
	}

	if err := query.Group("period").Order("period").Scan(&counts).Error; err != nil {
		return nil, err
	}

	return counts, nil
}
