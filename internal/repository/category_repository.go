package repository

import (
	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
)

type CategoryRepository interface {
	Create(category *models.Category) error
	GetByID(id uint) (*models.Category, error)
	GetAll() ([]models.Category, error)
	Update(category *models.Category) error
	Delete(id uint) error
	GetBySlug(slug string) (*models.Category, error)
	GetWithPostCount() ([]models.Category, error)
	ExistsBySlug(slug string) (bool, error)
}

type categoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) Create(category *models.Category) error {
	return r.db.Create(category).Error
}

func (r *categoryRepository) GetByID(id uint) (*models.Category, error) {
	var category models.Category
	err := r.db.Preload("Posts").First(&category, id).Error
	return &category, err
}

func (r *categoryRepository) GetAll() ([]models.Category, error) {
	var categories []models.Category
	err := r.db.Find(&categories).Error
	return categories, err
}

func (r *categoryRepository) Update(category *models.Category) error {
	return r.db.Save(category).Error
}

func (r *categoryRepository) Delete(id uint) error {
	return r.db.Unscoped().Delete(&models.Category{}, id).Error
}

func (r *categoryRepository) GetWithPostCount() ([]models.Category, error) {
	type CategoryWithCount struct {
		models.Category
		PostCount int `json:"post_count"`
	}

	var categories []CategoryWithCount
	err := r.db.Model(&models.Category{}).
		Select("categories.*, COUNT(posts.id) as post_count").
		Joins("LEFT JOIN posts ON posts.category_id = categories.id AND posts.published = true").
		Group("categories.id").
		Order("categories.name ASC").
		Scan(&categories).Error

	var result []models.Category
	for _, c := range categories {
		result = append(result, c.Category)
	}

	return result, err
}

func (r *categoryRepository) GetBySlug(slug string) (*models.Category, error) {
	var category models.Category
	err := r.db.Where("slug = ?", slug).First(&category).Error
	return &category, err
}

func (r *categoryRepository) ExistsBySlug(slug string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Category{}).Where("slug = ?", slug).Count(&count).Error
	return count > 0, err
}
