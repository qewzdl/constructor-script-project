package service

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/internal/theme"
	"constructor-script-backend/pkg/logger"
)

var (
	ErrThemeManagerUnavailable = errors.New("theme manager is not configured")
	ErrThemeNotFound           = errors.New("theme not found")
)

const SettingKeyActiveTheme = "site.theme"

type ThemeService struct {
	mu sync.Mutex

	settingRepo  repository.SettingRepository
	manager      *theme.Manager
	defaultTheme string
}

func NewThemeService(settingRepo repository.SettingRepository, manager *theme.Manager, defaultTheme string) *ThemeService {
	return &ThemeService{
		settingRepo:  settingRepo,
		manager:      manager,
		defaultTheme: strings.ToLower(strings.TrimSpace(defaultTheme)),
	}
}

func (s *ThemeService) List() ([]models.ThemeInfo, error) {
	if s.manager == nil {
		return nil, ErrThemeManagerUnavailable
	}

	themes := s.manager.List()
	activeSlug := ""
	if active := s.manager.Active(); active != nil {
		activeSlug = active.Slug
	}

	results := make([]models.ThemeInfo, 0, len(themes))
	for _, t := range themes {
		info := models.ThemeInfo{
			Slug:         t.Slug,
			Name:         t.Metadata.Name,
			Description:  t.Metadata.Description,
			Version:      t.Metadata.Version,
			Author:       t.Metadata.Author,
			PreviewImage: t.Metadata.PreviewImage,
			Active:       t.Slug == activeSlug,
		}
		results = append(results, info)
	}

	return results, nil
}

func (s *ThemeService) Activate(slug string) (models.ThemeInfo, error) {
	if s.manager == nil {
		return models.ThemeInfo{}, ErrThemeManagerUnavailable
	}

	cleaned := strings.ToLower(strings.TrimSpace(slug))
	if cleaned == "" {
		cleaned = s.defaultTheme
	}
	if cleaned == "" {
		return models.ThemeInfo{}, errors.New("no theme specified")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	themeCandidate, ok := s.manager.Resolve(cleaned)
	if !ok {
		return models.ThemeInfo{}, fmt.Errorf("%w: %s", ErrThemeNotFound, cleaned)
	}

	if err := s.manager.Activate(cleaned); err != nil {
		return models.ThemeInfo{}, err
	}

	if s.settingRepo != nil {
		if err := s.settingRepo.Set(SettingKeyActiveTheme, themeCandidate.Slug); err != nil {
			return models.ThemeInfo{}, err
		}
	}

	info := models.ThemeInfo{
		Slug:         themeCandidate.Slug,
		Name:         themeCandidate.Metadata.Name,
		Description:  themeCandidate.Metadata.Description,
		Version:      themeCandidate.Metadata.Version,
		Author:       themeCandidate.Metadata.Author,
		PreviewImage: themeCandidate.Metadata.PreviewImage,
		Active:       true,
	}

	return info, nil
}

func (s *ThemeService) Active() (models.ThemeInfo, error) {
	if s.manager == nil {
		return models.ThemeInfo{}, ErrThemeManagerUnavailable
	}
	active := s.manager.Active()
	if active == nil {
		return models.ThemeInfo{}, fmt.Errorf("%w: %s", ErrThemeNotFound, "")
	}
	info := models.ThemeInfo{
		Slug:         active.Slug,
		Name:         active.Metadata.Name,
		Description:  active.Metadata.Description,
		Version:      active.Metadata.Version,
		Author:       active.Metadata.Author,
		PreviewImage: active.Metadata.PreviewImage,
		Active:       true,
	}
	return info, nil
}

func (s *ThemeService) ActiveTheme() (*theme.Theme, error) {
	if s.manager == nil {
		return nil, ErrThemeManagerUnavailable
	}
	active := s.manager.Active()
	if active == nil {
		return nil, fmt.Errorf("%w: %s", ErrThemeNotFound, "")
	}
	return active, nil
}

func (s *ThemeService) ResolveActiveSlug() string {
	if s.manager == nil {
		return ""
	}
	if active := s.manager.Active(); active != nil {
		return active.Slug
	}

	if s.settingRepo != nil {
		if setting, err := s.settingRepo.Get(SettingKeyActiveTheme); err == nil && setting != nil {
			return strings.ToLower(strings.TrimSpace(setting.Value))
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error(err, "Failed to resolve active theme from settings", nil)
		}
	}

	return strings.ToLower(strings.TrimSpace(s.defaultTheme))
}
