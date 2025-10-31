package repository

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
)

type PluginRepository interface {
	List() ([]models.Plugin, error)
	GetBySlug(slug string) (*models.Plugin, error)
	Save(plugin *models.Plugin) error
}

type pluginRepository struct {
	db *gorm.DB
}

func NewPluginRepository(db *gorm.DB) PluginRepository {
	if db == nil {
		return nil
	}
	return &pluginRepository{db: db}
}

func (r *pluginRepository) List() ([]models.Plugin, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("plugin repository is not configured")
	}

	var plugins []models.Plugin
	if err := r.db.Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}

func (r *pluginRepository) GetBySlug(slug string) (*models.Plugin, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("plugin repository is not configured")
	}

	cleaned := strings.ToLower(strings.TrimSpace(slug))
	if cleaned == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var plugin models.Plugin
	if err := r.db.Where("slug = ?", cleaned).First(&plugin).Error; err != nil {
		return nil, err
	}
	return &plugin, nil
}

func (r *pluginRepository) Save(plugin *models.Plugin) error {
	if r == nil || r.db == nil {
		return errors.New("plugin repository is not configured")
	}
	if plugin == nil {
		return errors.New("plugin is required")
	}

	plugin.Slug = strings.ToLower(strings.TrimSpace(plugin.Slug))
	if plugin.Slug == "" {
		return errors.New("plugin slug is required")
	}

	if plugin.InstalledAt.IsZero() {
		plugin.InstalledAt = time.Now().UTC()
	}

	return r.db.Save(plugin).Error
}
