package repository

import (
	"time"

	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PageRepository interface {
	Create(page *models.Page) error
	Update(page *models.Page) error
	Delete(id uint) error
	GetByID(id uint) (*models.Page, error)
	GetBySlug(slug string) (*models.Page, error)
	GetBySlugAny(slug string) (*models.Page, error)
	GetByPath(path string) (*models.Page, error)
	GetByPathAny(path string) (*models.Page, error)
	GetAll() ([]models.Page, error)
	GetAllAdmin() ([]models.Page, error)
	ExistsBySlug(slug string) (bool, error)
	ExistsBySlugExceptID(slug string, excludeID uint) (bool, error)
	ExistsByPath(path string) (bool, error)
	ExistsByPathExceptID(path string, excludeID uint) (bool, error)
}

type pageRepository struct {
	db *gorm.DB
}

func NewPageRepository(db *gorm.DB) PageRepository {
	return &pageRepository{db: db}
}

func (r *pageRepository) Create(page *models.Page) error {
	return r.db.Create(page).Error
}

func (r *pageRepository) Update(page *models.Page) error {
	return r.db.Save(page).Error
}

func (r *pageRepository) Delete(id uint) error {
	return r.db.Unscoped().Delete(&models.Page{}, id).Error
}

func (r *pageRepository) GetByID(id uint) (*models.Page, error) {
	var page models.Page
	if err := r.db.First(&page, id).Error; err != nil {
		return nil, err
	}
	return &page, nil
}

func (r *pageRepository) GetBySlug(slug string) (*models.Page, error) {
	var page models.Page
	now := time.Now().UTC()

	if err := r.db.Where("slug = ? AND published = ?", slug, true).
		Where("publish_at IS NULL OR publish_at <= ?", now).
		First(&page).Error; err != nil {
		return nil, err
	}
	return &page, nil
}

func (r *pageRepository) GetBySlugAny(slug string) (*models.Page, error) {
	var page models.Page
	if err := r.db.Where("slug = ?", slug).First(&page).Error; err != nil {
		return nil, err
	}
	return &page, nil
}

func (r *pageRepository) GetByPath(path string) (*models.Page, error) {
	var page models.Page
	now := time.Now().UTC()

	if err := r.db.Where("path = ? AND published = ?", path, true).
		Where("publish_at IS NULL OR publish_at <= ?", now).
		First(&page).Error; err != nil {
		return nil, err
	}
	return &page, nil
}

func (r *pageRepository) GetByPathAny(path string) (*models.Page, error) {
	var page models.Page
	if err := r.db.Where("path = ?", path).First(&page).Error; err != nil {
		return nil, err
	}
	return &page, nil
}

func (r *pageRepository) GetAll() ([]models.Page, error) {
	var pages []models.Page
	now := time.Now().UTC()

	if err := r.db.Where("published = ?", true).
		Where("publish_at IS NULL OR publish_at <= ?", now).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "order"}}).
		Order("COALESCE(pages.publish_at, pages.created_at) DESC").
		Find(&pages).Error; err != nil {
		return nil, err
	}
	return pages, nil
}

func (r *pageRepository) GetAllAdmin() ([]models.Page, error) {
	var pages []models.Page
	if err := r.db.Order(clause.OrderByColumn{Column: clause.Column{Name: "order"}}).
		Order("pages.created_at DESC").Find(&pages).Error; err != nil {
		return nil, err
	}
	return pages, nil
}

func (r *pageRepository) ExistsBySlug(slug string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Page{}).Where("slug = ?", slug).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *pageRepository) ExistsBySlugExceptID(slug string, excludeID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Page{}).
		Where("slug = ? AND id <> ?", slug, excludeID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *pageRepository) ExistsByPath(path string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Page{}).Where("path = ?", path).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *pageRepository) ExistsByPathExceptID(path string, excludeID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Page{}).
		Where("path = ? AND id <> ?", path, excludeID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
