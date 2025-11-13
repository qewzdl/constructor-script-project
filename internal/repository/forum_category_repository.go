package repository

import (
	"strings"

	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
)

type ForumCategoryRepository interface {
	Create(category *models.ForumCategory) error
	Update(category *models.ForumCategory) error
	Delete(id uint) error
	GetByID(id uint) (*models.ForumCategory, error)
	GetBySlug(slug string) (*models.ForumCategory, error)
	ListAll() ([]models.ForumCategory, error)
	ListWithQuestionCount() ([]models.ForumCategory, error)
	ExistsBySlug(slug string) (bool, error)
	ExistsBySlugExcludingID(slug string, excludeID uint) (bool, error)
	ExistsByName(name string) (bool, error)
	ExistsByNameExcludingID(name string, excludeID uint) (bool, error)
	ClearCategoryAssignments(categoryID uint) error
}

type forumCategoryRepository struct {
	db *gorm.DB
}

func NewForumCategoryRepository(db *gorm.DB) ForumCategoryRepository {
	return &forumCategoryRepository{db: db}
}

func (r *forumCategoryRepository) Create(category *models.ForumCategory) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.Create(category).Error
}

func (r *forumCategoryRepository) Update(category *models.ForumCategory) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.Save(category).Error
}

func (r *forumCategoryRepository) Delete(id uint) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.Delete(&models.ForumCategory{}, id).Error
}

func (r *forumCategoryRepository) GetByID(id uint) (*models.ForumCategory, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	var category models.ForumCategory
	if err := r.db.First(&category, id).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *forumCategoryRepository) GetBySlug(slug string) (*models.ForumCategory, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	cleaned := strings.TrimSpace(slug)
	var category models.ForumCategory
	if err := r.db.Where("slug = ?", cleaned).First(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *forumCategoryRepository) ListAll() ([]models.ForumCategory, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	var categories []models.ForumCategory
	err := r.db.Order("name ASC").Find(&categories).Error
	return categories, err
}

func (r *forumCategoryRepository) ListWithQuestionCount() ([]models.ForumCategory, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	var categories []models.ForumCategory
	err := r.db.Model(&models.ForumCategory{}).
		Select("forum_categories.*, (SELECT COUNT(*) FROM forum_questions WHERE forum_questions.category_id = forum_categories.id) AS question_count").
		Order("name ASC").
		Find(&categories).Error
	return categories, err
}

func (r *forumCategoryRepository) ExistsBySlug(slug string) (bool, error) {
	if r == nil || r.db == nil {
		return false, gorm.ErrInvalidDB
	}
	cleaned := strings.TrimSpace(slug)
	if cleaned == "" {
		return false, nil
	}
	var count int64
	if err := r.db.Model(&models.ForumCategory{}).Where("slug = ?", cleaned).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *forumCategoryRepository) ExistsBySlugExcludingID(slug string, excludeID uint) (bool, error) {
	if r == nil || r.db == nil {
		return false, gorm.ErrInvalidDB
	}
	cleaned := strings.TrimSpace(slug)
	if cleaned == "" {
		return false, nil
	}
	var count int64
	if err := r.db.Model(&models.ForumCategory{}).Where("slug = ? AND id <> ?", cleaned, excludeID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *forumCategoryRepository) ExistsByName(name string) (bool, error) {
	if r == nil || r.db == nil {
		return false, gorm.ErrInvalidDB
	}
	cleaned := strings.TrimSpace(name)
	if cleaned == "" {
		return false, nil
	}
	var count int64
	if err := r.db.Model(&models.ForumCategory{}).
		Where("LOWER(name) = LOWER(?)", cleaned).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *forumCategoryRepository) ExistsByNameExcludingID(name string, excludeID uint) (bool, error) {
	if r == nil || r.db == nil {
		return false, gorm.ErrInvalidDB
	}
	cleaned := strings.TrimSpace(name)
	if cleaned == "" {
		return false, nil
	}
	var count int64
	if err := r.db.Model(&models.ForumCategory{}).
		Where("LOWER(name) = LOWER(?) AND id <> ?", cleaned, excludeID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *forumCategoryRepository) ClearCategoryAssignments(categoryID uint) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.Model(&models.ForumQuestion{}).Where("category_id = ?", categoryID).Update("category_id", nil).Error
}
