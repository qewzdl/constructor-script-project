package service

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/cache"
	"constructor-script-backend/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PageService struct {
	pageRepo repository.PageRepository
	cache    *cache.Cache
}

func normalizePagePath(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}

	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}

	cleaned := path.Clean(trimmed)
	if cleaned == "." {
		cleaned = "/"
	}
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	if cleaned != "/" && strings.HasSuffix(cleaned, "/") {
		cleaned = strings.TrimSuffix(cleaned, "/")
	}
	if strings.ContainsAny(cleaned, " \t\n\r") {
		return "", errors.New("page path cannot contain spaces")
	}

	return cleaned, nil
}

func defaultPathFromSlug(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "/"
	}
	if slug == "home" {
		return "/"
	}
	return "/" + slug
}

func (s *PageService) cachePage(page *models.Page) {
	if s == nil || s.cache == nil || page == nil {
		return
	}

	s.cache.Set(fmt.Sprintf("page:%d", page.ID), page, 1*time.Hour)

	if page.Slug != "" {
		s.cache.Set(fmt.Sprintf("page:slug:%s", page.Slug), page, 1*time.Hour)
	}

	if page.Path != "" {
		s.cache.Set(fmt.Sprintf("page:path:%s", page.Path), page, 1*time.Hour)
	}
}

func NewPageService(pageRepo repository.PageRepository, cacheService *cache.Cache) *PageService {
	return &PageService{
		pageRepo: pageRepo,
		cache:    cacheService,
	}
}

func (s *PageService) Create(req models.CreatePageRequest) (*models.Page, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, errors.New("page title is required")
	}

	var slug string
	if strings.TrimSpace(req.Slug) != "" {
		slug = utils.GenerateSlug(req.Slug)
	} else {
		slug = utils.GenerateSlug(req.Title)
	}

	if slug == "" {
		return nil, errors.New("page slug is required")
	}

	normalizedPath, err := normalizePagePath(req.Path)
	if err != nil {
		return nil, err
	}
	if normalizedPath == "" {
		normalizedPath = defaultPathFromSlug(slug)
	}

	exists, err := s.pageRepo.ExistsBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check page existence: %w", err)
	}
	if exists {
		return nil, errors.New("page with this title already exists")
	}

	existsByPath, err := s.pageRepo.ExistsByPath(normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check page path existence: %w", err)
	}
	if existsByPath {
		return nil, errors.New("page with this path already exists")
	}

	sections, err := s.prepareSections(req.Sections)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare sections: %w", err)
	}

	page := &models.Page{
		Title:       strings.TrimSpace(req.Title),
		Slug:        slug,
		Path:        normalizedPath,
		Description: req.Description,
		FeaturedImg: req.FeaturedImg,
		Published:   req.Published,
		Content:     strings.TrimSpace(req.Content),
		Sections:    sections,
		Template:    s.getTemplate(req.Template),
		HideHeader:  req.HideHeader,
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

func (s *PageService) ApplyDefinition(req models.CreatePageRequest) (*models.Page, error) {
	if s == nil || s.pageRepo == nil {
		return nil, errors.New("page repository not configured")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" && strings.TrimSpace(req.Slug) == "" {
		return nil, errors.New("page title is required")
	}
	req.Title = title

	sourceSlug := strings.TrimSpace(req.Slug)
	if sourceSlug == "" {
		sourceSlug = title
	}

	slug := utils.GenerateSlug(sourceSlug)
	if slug == "" {
		return nil, errors.New("page slug is required")
	}
	req.Slug = slug

	normalizedPath, err := normalizePagePath(req.Path)
	if err != nil {
		return nil, err
	}
	if normalizedPath == "" {
		normalizedPath = defaultPathFromSlug(slug)
	}
	req.Path = normalizedPath

	if err := s.removeExistingPages(slug, normalizedPath); err != nil {
		return nil, err
	}

	return s.Create(req)
}

func (s *PageService) removeExistingPages(slug, path string) error {
	if s == nil || s.pageRepo == nil {
		return errors.New("page repository not configured")
	}

	seen := make(map[uint]struct{})

	if path != "" {
		for {
			existing, err := s.pageRepo.GetByPathAny(path)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					break
				}
				return fmt.Errorf("failed to look up existing page by path: %w", err)
			}

			if _, ok := seen[existing.ID]; ok {
				break
			}
			seen[existing.ID] = struct{}{}

			if err := s.removePage(existing); err != nil {
				return err
			}
		}
	}

	for {
		existing, err := s.pageRepo.GetBySlugAny(slug)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return fmt.Errorf("failed to look up existing page: %w", err)
		}

		if _, ok := seen[existing.ID]; ok {
			break
		}
		seen[existing.ID] = struct{}{}

		if err := s.removePage(existing); err != nil {
			return err
		}
	}

	return nil
}

func (s *PageService) removePage(existing *models.Page) error {
	if existing == nil {
		return nil
	}

	if err := s.pageRepo.Delete(existing.ID); err != nil {
		return fmt.Errorf("failed to remove existing page: %w", err)
	}

	if s.cache != nil {
		s.cache.InvalidatePage(existing.ID)
		s.cache.Delete("pages:all")
		if existing.Path != "" {
			s.cache.Delete(fmt.Sprintf("page:path:%s", existing.Path))
		}
	}

	return nil
}

func (s *PageService) Update(id uint, req models.UpdatePageRequest) (*models.Page, error) {
	page, err := s.pageRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	originalSlug := page.Slug
	originalPath := page.Path
	originalPublished := page.Published
	slugChanged := false
	pathChanged := false

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return nil, errors.New("page title is required")
		}

		slug := utils.GenerateSlug(title)
		if slug == "" {
			return nil, errors.New("page slug is required")
		}

		page.Title = title
		if slug != originalSlug {
			slugChanged = true
		}
		page.Slug = slug
	}
	if req.Path != nil {
		normalizedPath, err := normalizePagePath(*req.Path)
		if err != nil {
			return nil, err
		}
		if normalizedPath == "" {
			normalizedPath = defaultPathFromSlug(page.Slug)
		}
		if normalizedPath != page.Path {
			pathChanged = true
		}
		page.Path = normalizedPath
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
	if req.HideHeader != nil {
		page.HideHeader = *req.HideHeader
	}
	if req.Order != nil {
		page.Order = *req.Order
	}
	if req.Content != nil {
		page.Content = strings.TrimSpace(*req.Content)
	}

	if req.Sections != nil {
		sections, err := s.prepareSections(*req.Sections)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare sections: %w", err)
		}
		page.Sections = sections
	}

	shouldValidateSlug := slugChanged || (!originalPublished && page.Published)
	shouldValidatePath := pathChanged || (!originalPublished && page.Published)

	if shouldValidateSlug {
		exists, err := s.pageRepo.ExistsBySlugExceptID(page.Slug, page.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check page existence: %w", err)
		}

		if exists {
			return nil, errors.New("page with this title already exists")
		}
	}

	if shouldValidatePath {
		exists, err := s.pageRepo.ExistsByPathExceptID(page.Path, page.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check page path existence: %w", err)
		}

		if exists {
			return nil, errors.New("page with this path already exists")
		}
	}

	if err := s.pageRepo.Update(page); err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cache.InvalidatePage(id)
		s.cache.Delete("pages:all")
		if originalPath != "" {
			s.cache.Delete(fmt.Sprintf("page:path:%s", originalPath))
		}
		if page.Path != "" {
			s.cache.Delete(fmt.Sprintf("page:path:%s", page.Path))
		}
	}

	return s.pageRepo.GetByID(page.ID)
}

func (s *PageService) Delete(id uint) error {
	page, err := s.pageRepo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.pageRepo.Delete(id); err != nil {
		return err
	}

	if s.cache != nil {
		s.cache.InvalidatePage(id)
		s.cache.Delete("pages:all")
		if page.Path != "" {
			s.cache.Delete(fmt.Sprintf("page:path:%s", page.Path))
		}
	}

	return nil
}

func (s *PageService) GetByID(id uint) (*models.Page, error) {
	if s.cache != nil {
		var page models.Page
		cacheKey := fmt.Sprintf("page:%d", id)
		if err := s.cache.Get(cacheKey, &page); err == nil {
			if !page.Published {
				return nil, gorm.ErrRecordNotFound
			}
			return &page, nil
		}
	}

	page, err := s.pageRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !page.Published {
		return nil, gorm.ErrRecordNotFound
	}

	if s.cache != nil {
		s.cachePage(page)
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
		s.cachePage(page)
	}

	return page, nil
}

func (s *PageService) GetByPath(requestedPath string) (*models.Page, error) {
	normalizedPath, err := normalizePagePath(requestedPath)
	if err != nil {
		return nil, err
	}
	if normalizedPath == "" {
		normalizedPath = "/"
	}

	if s.cache != nil {
		var page models.Page
		cacheKey := fmt.Sprintf("page:path:%s", normalizedPath)
		if err := s.cache.Get(cacheKey, &page); err == nil {
			return &page, nil
		}
	}

	page, err := s.pageRepo.GetByPath(normalizedPath)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cachePage(page)
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
		if page.Path != "" {
			s.cache.Delete(fmt.Sprintf("page:path:%s", page.Path))
		}
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
		if page.Path != "" {
			s.cache.Delete(fmt.Sprintf("page:path:%s", page.Path))
		}
	}

	return nil
}

func (s *PageService) prepareSections(sections []models.Section) (models.PostSections, error) {
	if len(sections) == 0 {
		return models.PostSections{}, nil
	}

	prepared := make(models.PostSections, 0, len(sections))

	for i, section := range sections {
		sectionType := strings.TrimSpace(section.Type)
		sectionType = strings.ToLower(sectionType)
		if sectionType == "" {
			sectionType = "standard"
		}

		switch sectionType {
		case "standard":
			if len(section.Elements) > 0 {
				preparedElements, err := s.prepareSectionElements(section.Elements)
				if err != nil {
					return nil, fmt.Errorf("section %d: %w", i, err)
				}
				section.Elements = preparedElements
			}
		case "hero":
			section.Elements = nil
		case "posts_list":
			section.Elements = nil
			limit := section.Limit
			if limit <= 0 {
				limit = constants.DefaultPostListSectionLimit
			}
			if limit > constants.MaxPostListSectionLimit {
				limit = constants.MaxPostListSectionLimit
			}
			section.Limit = limit
		default:
			return nil, fmt.Errorf("section %d: unknown type '%s'", i, sectionType)
		}

		if section.Title == "" {
			return nil, fmt.Errorf("section %d: title is required", i)
		}

		if section.ID == "" {
			section.ID = uuid.New().String()
		}

		if section.Order == 0 {
			section.Order = i + 1
		}

		section.Type = sectionType

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
		case "paragraph", "image", "image_group", "list", "search":

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
