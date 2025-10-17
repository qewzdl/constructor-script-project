package repository

import (
	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
)

type TagRepository interface {
	Create(tag *models.Tag) error
	Delete(id uint) error
	GetByID(id uint) (*models.Tag, error)
	GetBySlug(slug string) (*models.Tag, error)
	GetAll() ([]models.Tag, error)
	GetUsed() ([]models.Tag, error)
	ExistsByName(name string) (bool, error)
}

type tagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

func (r *tagRepository) Create(tag *models.Tag) error {
	return r.db.Create(tag).Error
}

func (r *tagRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM post_tags WHERE tag_id = ?", id).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&models.Tag{}, id).Error
	})
}

func (r *tagRepository) GetByID(id uint) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.First(&tag, id).Error
	return &tag, err
}

func (r *tagRepository) GetBySlug(slug string) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.Where("slug = ?", slug).First(&tag).Error
	return &tag, err
}

func (r *tagRepository) GetAll() ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Find(&tags).Error
	return tags, err
}

func (r *tagRepository) GetUsed() ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Model(&models.Tag{}).
		Select("tags.*").
		Joins("JOIN post_tags ON post_tags.tag_id = tags.id").
		Group("tags.id").
		Order("LOWER(tags.name)").
		Find(&tags).Error
	return tags, err
}

func (r *tagRepository) GetPopular(limit int) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Raw(`
		SELECT tags.*, COUNT(post_tags.post_id) as post_count
		FROM tags
		LEFT JOIN post_tags ON post_tags.tag_id = tags.id
		GROUP BY tags.id
		ORDER BY post_count DESC
		LIMIT ?
	`, limit).Scan(&tags).Error
	return tags, err
}

func (r *tagRepository) GetWithPostCount() ([]models.Tag, error) {
	type TagWithCount struct {
		models.Tag
		PostCount int `json:"post_count"`
	}

	var tags []TagWithCount
	err := r.db.Model(&models.Tag{}).
		Select("tags.*, COUNT(post_tags.post_id) as post_count").
		Joins("LEFT JOIN post_tags ON post_tags.tag_id = tags.id").
		Group("tags.id").
		Order("post_count DESC").
		Scan(&tags).Error

	var result []models.Tag
	for _, t := range tags {
		result = append(result, t.Tag)
	}

	return result, err
}

func (r *tagRepository) ExistsByName(name string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Tag{}).Where("name = ?", name).Count(&count).Error
	return count > 0, err
}
