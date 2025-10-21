package repository

import (
	"strings"

	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
)

type MenuRepository interface {
	List() ([]models.MenuItem, error)
	Create(item *models.MenuItem) error
	Update(item *models.MenuItem) error
	Delete(id uint) error
	GetByID(id uint) (*models.MenuItem, error)
	NextOrder(location string) (int, error)
}

type menuRepository struct {
	db *gorm.DB
}

func NewMenuRepository(db *gorm.DB) MenuRepository {
	return &menuRepository{db: db}
}

func (r *menuRepository) List() ([]models.MenuItem, error) {
	var items []models.MenuItem
	err := r.db.Order("\"order\" ASC, id ASC").Find(&items).Error
	return items, err
}

func (r *menuRepository) Create(item *models.MenuItem) error {
	return r.db.Create(item).Error
}

func (r *menuRepository) Update(item *models.MenuItem) error {
	return r.db.Save(item).Error
}

func (r *menuRepository) Delete(id uint) error {
	return r.db.Delete(&models.MenuItem{}, id).Error
}

func (r *menuRepository) GetByID(id uint) (*models.MenuItem, error) {
	var item models.MenuItem
	err := r.db.First(&item, id).Error
	return &item, err
}

func (r *menuRepository) NextOrder(location string) (int, error) {
	var maxOrder int64
	query := r.db.Model(&models.MenuItem{}).Select("COALESCE(MAX(\"order\"), 0)")
	if strings.TrimSpace(location) != "" {
		query = query.Where("location = ?", location)
	}
	if err := query.Scan(&maxOrder).Error; err != nil {
		return 0, err
	}
	return int(maxOrder) + 1, nil
}
