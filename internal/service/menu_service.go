package service

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
)

const defaultMenuLocation = "header"

type MenuService struct {
	repo repository.MenuRepository
}

func NewMenuService(repo repository.MenuRepository) *MenuService {
	if repo == nil {
		return nil
	}
	return &MenuService{repo: repo}
}

func (s *MenuService) List() ([]models.MenuItem, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("menu repository not configured")
	}
	items, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	return models.NormalizeMenuItems(items), nil
}

func (s *MenuService) ListPublic() ([]models.MenuItem, error) {
	return s.List()
}

func (s *MenuService) Create(req models.CreateMenuItemRequest) (*models.MenuItem, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("menu repository not configured")
	}

	title := strings.TrimSpace(req.Title)
	url := strings.TrimSpace(req.URL)
	location := normalizeMenuLocation(req.Location)

	if title == "" {
		return nil, errors.New("title is required")
	}
	if url == "" {
		return nil, errors.New("url is required")
	}

	order := 0
	if req.Order != nil {
		order = *req.Order
	} else {
		nextOrder, err := s.repo.NextOrder(location)
		if err != nil {
			return nil, err
		}
		order = nextOrder
	}

	item := &models.MenuItem{
		Title:    title,
		Label:    title,
		URL:      url,
		Location: location,
		Order:    order,
	}
	item.EnsureTextFields()

	if err := s.repo.Create(item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *MenuService) Update(id uint, req models.UpdateMenuItemRequest) (*models.MenuItem, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("menu repository not configured")
	}

	item, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	item.Location = normalizeMenuLocation(item.Location)
	item.EnsureTextFields()

	title := strings.TrimSpace(req.Title)
	url := strings.TrimSpace(req.URL)

	if title == "" {
		return nil, errors.New("title is required")
	}
	if url == "" {
		return nil, errors.New("url is required")
	}

	item.Title = title
	item.Label = title
	item.URL = url
	if req.Order != nil {
		item.Order = *req.Order
	}
	if req.Location != nil {
		item.Location = normalizeMenuLocation(*req.Location)
	}
	item.EnsureTextFields()

	if err := s.repo.Update(item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *MenuService) Delete(id uint) error {
	if s == nil || s.repo == nil {
		return errors.New("menu repository not configured")
	}
	return s.repo.Delete(id)
}

func (s *MenuService) Reorder(orders []models.MenuOrder) error {
	if s == nil || s.repo == nil {
		return errors.New("menu repository not configured")
	}

	if len(orders) == 0 {
		return nil
	}

	for _, entry := range orders {
		if entry.ID == 0 {
			continue
		}

		item, err := s.repo.GetByID(entry.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		item.Order = entry.Order
		if err := s.repo.Update(item); err != nil {
			return err
		}
	}

	return nil
}

func normalizeMenuLocation(location string) string {
	cleaned := strings.ToLower(strings.TrimSpace(location))
	if cleaned == "" {
		return defaultMenuLocation
	}
	return cleaned
}
