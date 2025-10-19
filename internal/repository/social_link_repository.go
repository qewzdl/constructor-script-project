package repository

import (
	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
)

type SocialLinkRepository interface {
	List() ([]models.SocialLink, error)
	Create(link *models.SocialLink) error
	Update(link *models.SocialLink) error
	Delete(id uint) error
	GetByID(id uint) (*models.SocialLink, error)
	NextOrder() (int, error)
}

type socialLinkRepository struct {
	db *gorm.DB
}

func NewSocialLinkRepository(db *gorm.DB) SocialLinkRepository {
	return &socialLinkRepository{db: db}
}

func (r *socialLinkRepository) List() ([]models.SocialLink, error) {
	var links []models.SocialLink
	err := r.db.Order("\"order\" ASC, id ASC").Find(&links).Error
	return links, err
}

func (r *socialLinkRepository) Create(link *models.SocialLink) error {
	return r.db.Create(link).Error
}

func (r *socialLinkRepository) Update(link *models.SocialLink) error {
	return r.db.Save(link).Error
}

func (r *socialLinkRepository) Delete(id uint) error {
	return r.db.Delete(&models.SocialLink{}, id).Error
}

func (r *socialLinkRepository) GetByID(id uint) (*models.SocialLink, error) {
	var link models.SocialLink
	err := r.db.First(&link, id).Error
	return &link, err
}

func (r *socialLinkRepository) NextOrder() (int, error) {
	var maxOrder int64
	if err := r.db.Model(&models.SocialLink{}).Select("COALESCE(MAX(\"order\"), 0)").Scan(&maxOrder).Error; err != nil {
		return 0, err
	}
	return int(maxOrder) + 1, nil
}
