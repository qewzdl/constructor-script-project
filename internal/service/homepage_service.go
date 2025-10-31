package service

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrHomepagePageNotPublished = errors.New("page must be published before it can be set as the homepage")
	ErrHomepagePageScheduled    = errors.New("scheduled pages cannot be set as the homepage until published")
)

const settingKeySiteHomepage = "site.homepage_page_id"

type HomepageService struct {
	settingRepo repository.SettingRepository
	pageRepo    repository.PageRepository
}

func NewHomepageService(settingRepo repository.SettingRepository, pageRepo repository.PageRepository) *HomepageService {
	return &HomepageService{
		settingRepo: settingRepo,
		pageRepo:    pageRepo,
	}
}

func (s *HomepageService) ListOptions() ([]models.HomepagePage, error) {
	if s == nil || s.pageRepo == nil {
		return nil, errors.New("page repository not configured")
	}

	pages, err := s.pageRepo.GetAllAdmin()
	if err != nil {
		return nil, err
	}

	options := make([]models.HomepagePage, 0, len(pages))
	for _, page := range pages {
		options = append(options, toHomepagePage(&page))
	}

	sort.SliceStable(options, func(i, j int) bool {
		if options[i].UpdatedAt.Equal(options[j].UpdatedAt) {
			return strings.ToLower(options[i].Title) < strings.ToLower(options[j].Title)
		}
		return options[i].UpdatedAt.After(options[j].UpdatedAt)
	})

	return options, nil
}

func (s *HomepageService) GetSelection() (*models.HomepagePage, error) {
	page, err := s.getStoredPage()
	if err != nil {
		return nil, err
	}
	if page == nil {
		return nil, nil
	}
	info := toHomepagePage(page)
	return &info, nil
}

func (s *HomepageService) SetHomepage(pageID uint) (*models.HomepagePage, error) {
	if s == nil || s.pageRepo == nil {
		return nil, errors.New("page repository not configured")
	}
	if s.settingRepo == nil {
		return nil, errors.New("setting repository not configured")
	}

	page, err := s.pageRepo.GetByID(pageID)
	if err != nil {
		return nil, err
	}

	if !page.Published {
		return nil, ErrHomepagePageNotPublished
	}
	if page.PublishAt != nil {
		now := time.Now().UTC()
		if page.PublishAt.After(now) {
			return nil, ErrHomepagePageScheduled
		}
	}

	if err := s.settingRepo.Set(settingKeySiteHomepage, strconv.FormatUint(uint64(pageID), 10)); err != nil {
		return nil, err
	}

	info := toHomepagePage(page)
	return &info, nil
}

func (s *HomepageService) ClearHomepage() error {
	if s == nil || s.settingRepo == nil {
		return errors.New("setting repository not configured")
	}
	if err := s.settingRepo.Delete(settingKeySiteHomepage); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func (s *HomepageService) GetActiveHomepage() (*models.Page, error) {
	page, err := s.getStoredPage()
	if err != nil {
		return nil, err
	}
	if page == nil {
		return nil, nil
	}

	if !page.Published {
		return nil, nil
	}
	if page.PublishAt != nil {
		now := time.Now().UTC()
		if page.PublishAt.After(now) {
			return nil, nil
		}
	}

	return page, nil
}

func (s *HomepageService) getStoredPage() (*models.Page, error) {
	if s == nil || s.settingRepo == nil {
		return nil, errors.New("setting repository not configured")
	}
	if s.pageRepo == nil {
		return nil, errors.New("page repository not configured")
	}

	setting, err := s.settingRepo.Get(settingKeySiteHomepage)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	value := strings.TrimSpace(setting.Value)
	if value == "" {
		return nil, nil
	}

	id, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		_ = s.settingRepo.Delete(settingKeySiteHomepage)
		return nil, fmt.Errorf("invalid homepage setting value: %w", err)
	}

	page, err := s.pageRepo.GetByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = s.settingRepo.Delete(settingKeySiteHomepage)
			return nil, nil
		}
		return nil, err
	}

	return page, nil
}

func toHomepagePage(page *models.Page) models.HomepagePage {
	if page == nil {
		return models.HomepagePage{}
	}

	return models.HomepagePage{
		ID:        page.ID,
		Title:     page.Title,
		Slug:      page.Slug,
		Path:      page.Path,
		Published: page.Published,
		PublishAt: page.PublishAt,
		UpdatedAt: page.UpdatedAt,
		CreatedAt: page.CreatedAt,
	}
}
