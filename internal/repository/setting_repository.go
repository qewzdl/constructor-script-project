package repository

import (
	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingRepository interface {
	Get(key string) (*models.Setting, error)
	Set(key, value string) error
	Delete(key string) error
}

type settingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) SettingRepository {
	return &settingRepository{db: db}
}

func (r *settingRepository) Get(key string) (*models.Setting, error) {
	var setting models.Setting
	err := r.db.First(&setting, "key = ?", key).Error
	return &setting, err
}

func (r *settingRepository) Set(key, value string) error {
	setting := &models.Setting{Key: key, Value: value}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"value": value}),
	}).Create(setting).Error
}

func (r *settingRepository) Delete(key string) error {
	return r.db.Unscoped().Delete(&models.Setting{}, "key = ?", key).Error
}
