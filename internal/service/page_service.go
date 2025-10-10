package service

import (
	"errors"
	"fmt"
	"time"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/cache"
	"constructor-script-backend/pkg/utils"

	"github.com/google/uuid"
)

type PageService struct {
	pageRepo repository.PageRepository
	cache    *cache.Cache
}

func NewPageService(pageRepo repository.PageRepository, cacheService *cache.Cache) *PageService {
	return &PageService{
		pageRepo: pageRepo,
		cache:    cacheService,
	}
}

func (s *PageService) Create(req models.CreatePageRequest) (*models.Page, error) {
	if req.Title == "" {
		return nil, errors.New("page title is required")
	}

	slug := utils.GenerateSlug(req.Title)

	exists, err := s.pageRepo.ExistsBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check page existence: %w", err)
	}
	if exists {
		return nil, errors.New("page with this title already exists")
	}

	sections, err := s.prepareSections(req.Sections)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare sections: %w", err)
	}

	page := &models.Page{
		Title:       req.Title,
		Slug:        slug,
		Description: req.Description,
		FeaturedImg: req.FeaturedImg,
		Published:   req.Published,
		Sections:    sections,
		Template:    s.getTemplate(req.Template),
		Order:       req.Order,
	}

	if err := s.pageRepo.Create(page); err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	if s.cache != nil {
		s.cache.Delete("pages:all")
	}

	return s.pageRepo.GetByID(page.ID)
}

func (s *PageService) Update(id uint, req models.UpdatePageRequest) (*models.Page, error) {
	page, err := s.pageRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		page.Title = *req.Title
		page.Slug = utils.GenerateSlug(*req.Title)
	}
	if req.Description != nil {
		page.Description = *req.Description
	}
	if req.FeaturedImg != nil {
		page.FeaturedImg = *req.FeaturedImg
	}
	if req.Published != nil {
		page.Published = *req.Published
	}
	if req.Template != nil {
		page.Template = s.getTemplate(*req.Template)
	}
	if req.Order != nil {
		page.Order = *req.Order
	}

	if req.Sections != nil {
		sections, err := s.prepareSections(*req.Sections)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare sections: %w", err)
		}
		page.Sections = sections
	}

	if err := s.pageRepo.Update(page); err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cache.InvalidatePage(id)
		s.cache.Delete("pages:all")
	}

	return s.pageRepo.GetByID(page.ID)
}

func (s *PageService) Delete(id uint) error {
	if err := s.pageRepo.Delete(id); err != nil {
		return err
	}

	if s.cache != nil {
		s.cache.InvalidatePage(id)
		s.cache.Delete("pages:all")
	}

	return nil
}

func (s *PageService) GetByID(id uint) (*models.Page, error) {
	if s.cache != nil {
		var page models.Page
		cacheKey := fmt.Sprintf("page:%d", id)
		if err := s.cache.Get(cacheKey, &page); err == nil {
			return &page, nil
		}
	}

	page, err := s.pageRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		cacheKey := fmt.Sprintf("page:%d", id)
		s.cache.Set(cacheKey, page, 1*time.Hour)
	}

	return page, nil
}

func (s *PageService) GetBySlug(slug string) (*models.Page, error) {
	if s.cache != nil {
		var page models.Page
		cacheKey := fmt.Sprintf("page:slug:%s", slug)
		if err := s.cache.Get(cacheKey, &page); err == nil {
			return &page, nil
		}
	}

	page, err := s.pageRepo.GetBySlug(slug)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		cacheKey := fmt.Sprintf("page:slug:%s", slug)
		s.cache.Set(cacheKey, page, 1*time.Hour)
		s.cache.Set(fmt.Sprintf("page:%d", page.ID), page, 1*time.Hour)
	}

	return page, nil
}

func (s *PageService) GetAll() ([]models.Page, error) {
	if s.cache != nil {
		var pages []models.Page
		if err := s.cache.Get("pages:all", &pages); err == nil {
			return pages, nil
		}
	}

	pages, err := s.pageRepo.GetAll()
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cache.Set("pages:all", pages, 1*time.Hour)
	}

	return pages, nil
}

func (s *PageService) GetAllAdmin() ([]models.Page, error) {
	return s.pageRepo.GetAllAdmin()
}

func (s *PageService) PublishPage(id uint) error {
	page, err := s.pageRepo.GetByID(id)
	if err != nil {
		return err
	}

	page.Published = true

	if err := s.pageRepo.Update(page); err != nil {
		return err
	}

	if s.cache != nil {
		s.cache.InvalidatePage(id)
		s.cache.Delete("pages:all")
	}

	return nil
}

func (s *PageService) UnpublishPage(id uint) error {
	page, err := s.pageRepo.GetByID(id)
	if err != nil {
		return err
	}

	page.Published = false

	if err := s.pageRepo.Update(page); err != nil {
		return err
	}

	if s.cache != nil {
		s.cache.InvalidatePage(id)
		s.cache.Delete("pages:all")
	}

	return nil
}

func (s *PageService) prepareSections(sections []models.Section) (models.PostSections, error) {
	if len(sections) == 0 {
		return models.PostSections{}, nil
	}

	prepared := make(models.PostSections, 0, len(sections))

	for i, section := range sections {
		if section.Title == "" {
			return nil, fmt.Errorf("section %d: title is required", i)
		}

		if section.ID == "" {
			section.ID = uuid.New().String()
		}

		if section.Order == 0 {
			section.Order = i + 1
		}

		if len(section.Elements) > 0 {
			preparedElements, err := s.prepareSectionElements(section.Elements)
			if err != nil {
				return nil, fmt.Errorf("section %d: %w", i, err)
			}
			section.Elements = preparedElements
		}

		prepared = append(prepared, section)
	}

	return prepared, nil
}

func (s *PageService) prepareSectionElements(elements []models.SectionElement) ([]models.SectionElement, error) {
	prepared := make([]models.SectionElement, 0, len(elements))

	for i, elem := range elements {
		if elem.ID == "" {
			elem.ID = uuid.New().String()
		}

		if elem.Order == 0 {
			elem.Order = i + 1
		}

		switch elem.Type {
		case "paragraph", "image", "image_group":

		default:
			return nil, fmt.Errorf("element %d: unknown type '%s'", i, elem.Type)
		}

		if elem.Content == nil {
			return nil, fmt.Errorf("element %d: content is required", i)
		}

		prepared = append(prepared, elem)
	}

	return prepared, nil
}

func (s *PageService) getTemplate(template string) string {
	if template == "" {
		return "page"
	}
	return template
}
