package repository

import (
	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
)

type ForumAnswerRepository interface {
	Create(answer *models.ForumAnswer) error
	Update(answer *models.ForumAnswer) error
	Delete(id uint) error
	GetByID(id uint) (*models.ForumAnswer, error)
	ListByQuestion(questionID uint) ([]models.ForumAnswer, error)
}

type forumAnswerRepository struct {
	db *gorm.DB
}

func NewForumAnswerRepository(db *gorm.DB) ForumAnswerRepository {
	return &forumAnswerRepository{db: db}
}

func (r *forumAnswerRepository) Create(answer *models.ForumAnswer) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.Create(answer).Error
}

func (r *forumAnswerRepository) Update(answer *models.ForumAnswer) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.Save(answer).Error
}

func (r *forumAnswerRepository) Delete(id uint) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.Delete(&models.ForumAnswer{}, id).Error
}

func (r *forumAnswerRepository) GetByID(id uint) (*models.ForumAnswer, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	var answer models.ForumAnswer
	err := r.db.Preload("Author").First(&answer, id).Error
	return &answer, err
}

func (r *forumAnswerRepository) ListByQuestion(questionID uint) ([]models.ForumAnswer, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	var answers []models.ForumAnswer
	err := r.db.Where("question_id = ?", questionID).
		Preload("Author").
		Order("rating DESC, created_at ASC").
		Find(&answers).Error
	return answers, err
}
