package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/utils"
)

type CategoryService struct {
	categoryRepo repository.ForumCategoryRepository
}

func NewCategoryService(categoryRepo repository.ForumCategoryRepository) *CategoryService {
	svc := &CategoryService{}
	svc.SetRepository(categoryRepo)
	return svc
}

func (s *CategoryService) SetRepository(categoryRepo repository.ForumCategoryRepository) {
	if s == nil {
		return
	}
	s.categoryRepo = categoryRepo
}

func (s *CategoryService) List(includeCounts bool) ([]models.ForumCategory, error) {
	if s == nil || s.categoryRepo == nil {
		return nil, errors.New("category repository not configured")
	}
	if includeCounts {
		return s.categoryRepo.ListWithQuestionCount()
	}
	return s.categoryRepo.ListAll()
}

func (s *CategoryService) GetAll() ([]models.ForumCategory, error) {
	return s.List(false)
}

func (s *CategoryService) GetByID(id uint) (*models.ForumCategory, error) {
	if s == nil || s.categoryRepo == nil {
		return nil, errors.New("category repository not configured")
	}
	category, err := s.categoryRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return category, nil
}

func (s *CategoryService) Create(req models.CreateForumCategoryRequest) (*models.ForumCategory, error) {
	if s == nil || s.categoryRepo == nil {
		return nil, errors.New("category repository not configured")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("category name is required")
	}
	slug, err := s.generateUniqueSlug(name, 0)
	if err != nil {
		return nil, err
	}
	category := &models.ForumCategory{
		Name: name,
		Slug: slug,
	}
	if err := s.categoryRepo.Create(category); err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}
	return category, nil
}

func (s *CategoryService) Update(id uint, req models.UpdateForumCategoryRequest) (*models.ForumCategory, error) {
	if s == nil || s.categoryRepo == nil {
		return nil, errors.New("category repository not configured")
	}
	category, err := s.categoryRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.New("category name cannot be empty")
		}
		if !strings.EqualFold(name, category.Name) {
			slug, slugErr := s.generateUniqueSlug(name, id)
			if slugErr != nil {
				return nil, slugErr
			}
			category.Name = name
			category.Slug = slug
		} else {
			category.Name = name
		}
	}
	if err := s.categoryRepo.Update(category); err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}
	return category, nil
}

func (s *CategoryService) Delete(id uint) error {
	if s == nil || s.categoryRepo == nil {
		return errors.New("category repository not configured")
	}
	if _, err := s.categoryRepo.GetByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCategoryNotFound
		}
		return err
	}
	if err := s.categoryRepo.ClearCategoryAssignments(id); err != nil {
		return fmt.Errorf("failed to clear category assignments: %w", err)
	}
	if err := s.categoryRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	return nil
}

func (s *CategoryService) generateUniqueSlug(name string, excludeID uint) (string, error) {
	if s == nil || s.categoryRepo == nil {
		return "", errors.New("category repository not configured")
	}
	base := utils.GenerateSlug(name)
	if base == "" {
		base = fmt.Sprintf("category-%d", time.Now().Unix())
	}
	slug := base
	for attempt := 1; attempt < 1000; attempt++ {
		var exists bool
		var err error
		if excludeID > 0 {
			exists, err = s.categoryRepo.ExistsBySlugExcludingID(slug, excludeID)
		} else {
			exists, err = s.categoryRepo.ExistsBySlug(slug)
		}
		if err != nil {
			return "", fmt.Errorf("failed to verify category slug: %w", err)
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, attempt)
	}
	return "", errors.New("failed to generate unique slug for category")
}
