package service

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
)

type SocialLinkService struct {
	repo repository.SocialLinkRepository
}

func NewSocialLinkService(repo repository.SocialLinkRepository) *SocialLinkService {
	if repo == nil {
		return nil
	}
	return &SocialLinkService{repo: repo}
}

func (s *SocialLinkService) List() ([]models.SocialLink, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("social link repository not configured")
	}
	return s.repo.List()
}

func (s *SocialLinkService) ListPublic() ([]models.SocialLink, error) {
	return s.List()
}

func (s *SocialLinkService) Create(req models.CreateSocialLinkRequest) (*models.SocialLink, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("social link repository not configured")
	}

	name := strings.TrimSpace(req.Name)
	url := strings.TrimSpace(req.URL)
	icon := strings.TrimSpace(req.Icon)

	if name == "" {
		return nil, errors.New("name is required")
	}
	if url == "" {
		return nil, errors.New("url is required")
	}

	order := 0
	if req.Order != nil {
		order = *req.Order
	} else {
		nextOrder, err := s.repo.NextOrder()
		if err != nil {
			return nil, err
		}
		order = nextOrder
	}

	link := &models.SocialLink{
		Name:  name,
		URL:   url,
		Icon:  icon,
		Order: order,
	}

	if err := s.repo.Create(link); err != nil {
		return nil, err
	}

	return link, nil
}

func (s *SocialLinkService) Update(id uint, req models.UpdateSocialLinkRequest) (*models.SocialLink, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("social link repository not configured")
	}

	link, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	url := strings.TrimSpace(req.URL)
	icon := strings.TrimSpace(req.Icon)

	if name == "" {
		return nil, errors.New("name is required")
	}
	if url == "" {
		return nil, errors.New("url is required")
	}

	link.Name = name
	link.URL = url
	link.Icon = icon
	if req.Order != nil {
		link.Order = *req.Order
	}

	if err := s.repo.Update(link); err != nil {
		return nil, err
	}

	return link, nil
}

func (s *SocialLinkService) Delete(id uint) error {
	if s == nil || s.repo == nil {
		return errors.New("social link repository not configured")
	}
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	return nil
}

func (s *SocialLinkService) Reorder(orders map[uint]int) error {
	if s == nil || s.repo == nil {
		return errors.New("social link repository not configured")
	}
	for id, order := range orders {
		link, err := s.repo.GetByID(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		link.Order = order
		if err := s.repo.Update(link); err != nil {
			return err
		}
	}
	return nil
}
