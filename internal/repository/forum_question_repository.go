package repository

import (
	"strings"

	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
)

type ForumQuestionRepository interface {
	Create(question *models.ForumQuestion) error
	Update(question *models.ForumQuestion) error
	Delete(id uint) error
	GetByID(id uint) (*models.ForumQuestion, error)
	GetBySlug(slug string) (*models.ForumQuestion, error)
	List(offset, limit int, search string, authorID *uint, categoryID *uint) ([]models.ForumQuestion, int64, error)
	ExistsBySlug(slug string) (bool, error)
	IncrementViews(id uint) error
}

type forumQuestionRepository struct {
	db *gorm.DB
}

func NewForumQuestionRepository(db *gorm.DB) ForumQuestionRepository {
	return &forumQuestionRepository{db: db}
}

func (r *forumQuestionRepository) Create(question *models.ForumQuestion) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.Create(question).Error
}

func (r *forumQuestionRepository) Update(question *models.ForumQuestion) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.Save(question).Error
}

func (r *forumQuestionRepository) Delete(id uint) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.Delete(&models.ForumQuestion{}, id).Error
}

func (r *forumQuestionRepository) GetByID(id uint) (*models.ForumQuestion, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	var question models.ForumQuestion
	err := r.db.
		Preload("Author").
		Preload("Category").
		Preload("Answers", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Author").Order("rating DESC, created_at ASC")
		}).
		First(&question, id).Error
	if err != nil {
		return nil, err
	}
	question.AnswersCount = len(question.Answers)
	return &question, nil
}

func (r *forumQuestionRepository) GetBySlug(slug string) (*models.ForumQuestion, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	cleaned := strings.TrimSpace(slug)
	var question models.ForumQuestion
	err := r.db.Where("slug = ?", cleaned).
		Preload("Author").
		Preload("Category").
		Preload("Answers", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Author").Order("rating DESC, created_at ASC")
		}).
		First(&question).Error
	if err != nil {
		return nil, err
	}
	question.AnswersCount = len(question.Answers)
	return &question, nil
}

func (r *forumQuestionRepository) List(offset, limit int, search string, authorID *uint, categoryID *uint) ([]models.ForumQuestion, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, gorm.ErrInvalidDB
	}

	query := r.db.Model(&models.ForumQuestion{}).
		Select("forum_questions.*, (SELECT COUNT(*) FROM forum_answers WHERE forum_answers.question_id = forum_questions.id) AS answers_count")

	cleanedSearch := strings.TrimSpace(search)
	if cleanedSearch != "" {
		like := "%" + cleanedSearch + "%"
		query = query.Where("title ILIKE ? OR content ILIKE ?", like, like)
	}

	if authorID != nil {
		query = query.Where("author_id = ?", *authorID)
	}

	if categoryID != nil {
		query = query.Where("category_id = ?", *categoryID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	var questions []models.ForumQuestion
	err := query.
		Preload("Author").
		Preload("Category").
		Order("rating DESC, created_at DESC").
		Find(&questions).Error
	return questions, total, err
}

func (r *forumQuestionRepository) ExistsBySlug(slug string) (bool, error) {
	if r == nil || r.db == nil {
		return false, gorm.ErrInvalidDB
	}
	cleaned := strings.TrimSpace(slug)
	if cleaned == "" {
		return false, nil
	}
	var count int64
	if err := r.db.Model(&models.ForumQuestion{}).Where("slug = ?", cleaned).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *forumQuestionRepository) IncrementViews(id uint) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.Model(&models.ForumQuestion{}).Where("id = ?", id).UpdateColumn("views", gorm.Expr("views + 1")).Error
}
