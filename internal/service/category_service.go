package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/cache"
	"constructor-script-backend/pkg/utils"

	"gorm.io/gorm"
)

type CategoryService struct {
	categoryRepo repository.CategoryRepository
	postRepo     repository.PostRepository
	cache        *cache.Cache
}

const (
	defaultCategoryName = "Uncategorized"
	defaultCategorySlug = "uncategorized"
)

func NewCategoryService(categoryRepo repository.CategoryRepository, postRepo repository.PostRepository, cacheService *cache.Cache) *CategoryService {
	return &CategoryService{
		categoryRepo: categoryRepo,
		postRepo:     postRepo,
		cache:        cacheService,
	}
}

func (s *CategoryService) EnsureDefaultCategory() (*models.Category, bool, error) {
	slug := defaultCategorySlug

	category, err := s.categoryRepo.GetBySlug(slug)
	if err == nil {
		return category, false, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, fmt.Errorf("failed to verify default category: %w", err)
	}

	created, createErr := s.Create(models.CreateCategoryRequest{
		Name:        defaultCategoryName,
		Description: "Default category for uncategorized posts",
	})
	if createErr != nil {
		category, fetchErr := s.categoryRepo.GetBySlug(slug)
		if fetchErr == nil {
			return category, false, nil
		}
		return nil, false, fmt.Errorf("failed to create default category: %w", createErr)
	}

	return created, true, nil
}

func (s *CategoryService) Create(req models.CreateCategoryRequest) (*models.Category, error) {

	if req.Name == "" {
		return nil, errors.New("category name is required")
	}

	slug := utils.GenerateSlug(req.Name)

	exists, err := s.categoryRepo.ExistsBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check category existence: %w", err)
	}
	if exists {
		return nil, errors.New("category with this name already exists")
	}

	category := &models.Category{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
	}

	if err := s.categoryRepo.Create(category); err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	if s.cache != nil {
		s.cache.Delete("categories:all")
		s.cache.Delete("categories:with_count")
	}

	return category, nil
}

func (s *CategoryService) GetByID(id uint) (*models.Category, error) {
	if s.cache != nil {
		var category models.Category
		if err := s.cache.GetCachedCategory(id, &category); err == nil {
			return &category, nil
		}
	}

	category, err := s.categoryRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cache.CacheCategory(id, category)
	}

	return category, nil
}

func (s *CategoryService) GetBySlug(slug string) (*models.Category, error) {
	if s.cache != nil {
		var category models.Category
		cacheKey := fmt.Sprintf("category:slug:%s", slug)
		if err := s.cache.Get(cacheKey, &category); err == nil {
			return &category, nil
		}
	}

	category, err := s.categoryRepo.GetBySlug(slug)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		cacheKey := fmt.Sprintf("category:slug:%s", slug)
		s.cache.Set(cacheKey, category, 2*time.Hour)
		s.cache.CacheCategory(category.ID, category)
	}

	return category, nil
}

func (s *CategoryService) GetAll() ([]models.Category, error) {
	if s.cache != nil {
		var categories []models.Category
		if err := s.cache.Get("categories:all", &categories); err == nil {
			return categories, nil
		}
	}

	categories, err := s.categoryRepo.GetAll()
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cache.Set("categories:all", categories, 2*time.Hour)
	}

	return categories, nil
}

func (s *CategoryService) GetWithPostCount() ([]models.Category, error) {
	if s.cache != nil {
		var categories []models.Category
		if err := s.cache.Get("categories:with_count", &categories); err == nil {
			return categories, nil
		}
	}

	categories, err := s.categoryRepo.GetWithPostCount()
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cache.Set("categories:with_count", categories, 30*time.Minute)
	}

	return categories, nil
}

func (s *CategoryService) Update(id uint, req models.CreateCategoryRequest) (*models.Category, error) {
	category, err := s.categoryRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	category.Name = req.Name
	category.Slug = utils.GenerateSlug(req.Name)
	category.Description = req.Description

	if err := s.categoryRepo.Update(category); err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cache.InvalidateCategory(id)
		s.cache.Delete("categories:all")
		s.cache.Delete("categories:with_count")

		oldSlug := category.Slug
		s.cache.Delete(fmt.Sprintf("category:slug:%s", oldSlug))
	}

	return category, nil
}

func (s *CategoryService) Delete(id uint) error {
	defaultCategory, _, err := s.EnsureDefaultCategory()
	if err != nil {
		return err
	}

	if defaultCategory != nil && id == defaultCategory.ID {
		return errors.New("default category cannot be deleted")
	}

	if s.postRepo != nil && defaultCategory != nil {
		if err := s.postRepo.ReassignCategory(id, defaultCategory.ID); err != nil {
			return fmt.Errorf("failed to reassign posts to default category: %w", err)
		}
	}

	if err := s.categoryRepo.Delete(id); err != nil {
		return err
	}

	if s.cache != nil {
		s.cache.InvalidateCategory(id)
		s.cache.Delete("categories:all")
		s.cache.Delete("categories:with_count")
		s.cache.InvalidatePostsCache()
	}

	return nil
}

func (s *CategoryService) GetCategoryStats(id uint) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("category:stats:%d", id)
	if s.cache != nil {
		var stats map[string]interface{}
		if err := s.cache.Get(cacheKey, &stats); err == nil {
			return stats, nil
		}
	}

	category, err := s.categoryRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"id":          category.ID,
		"name":        category.Name,
		"slug":        category.Slug,
		"post_count":  len(category.Posts),
		"description": category.Description,
	}

	if s.cache != nil {
		s.cache.Set(cacheKey, stats, 15*time.Minute)
	}

	return stats, nil
}

func (s *CategoryService) GetPopularCategories(limit int) ([]models.Category, error) {
	cacheKey := fmt.Sprintf("categories:popular:%d", limit)
	if s.cache != nil {
		var categories []models.Category
		if err := s.cache.Get(cacheKey, &categories); err == nil {
			return categories, nil
		}
	}

	allCategories, err := s.categoryRepo.GetWithPostCount()
	if err != nil {
		return nil, err
	}

	var categories []models.Category
	for i, cat := range allCategories {
		if i >= limit {
			break
		}
		categories = append(categories, cat)
	}

	if s.cache != nil {
		s.cache.Set(cacheKey, categories, 1*time.Hour)
	}

	return categories, nil
}

func (s *CategoryService) SearchCategories(query string) ([]models.Category, error) {
	allCategories, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	var result []models.Category
	query = strings.ToLower(strings.TrimSpace(query))

	for _, cat := range allCategories {
		if strings.Contains(strings.ToLower(cat.Name), query) ||
			strings.Contains(strings.ToLower(cat.Description), query) ||
			strings.Contains(strings.ToLower(cat.Slug), query) {
			result = append(result, cat)
		}
	}

	return result, nil
}

func (s *CategoryService) BulkCreate(requests []models.CreateCategoryRequest) ([]models.Category, error) {
	var categories []models.Category

	for _, req := range requests {
		category, err := s.Create(req)
		if err != nil {
			continue
		}
		categories = append(categories, *category)
	}

	if s.cache != nil {
		s.cache.Delete("categories:all")
		s.cache.Delete("categories:with_count")
	}

	return categories, nil
}

func (s *CategoryService) ReorderCategories(categoryIDs []uint) error {
	for index, id := range categoryIDs {
		category, err := s.categoryRepo.GetByID(id)
		if err != nil {
			return err
		}
		category.Order = index
		if err := s.categoryRepo.Update(category); err != nil {
			return err
		}
	}

	if s.cache != nil {
		s.cache.Delete("categories:all")
		s.cache.Delete("categories:with_count")
	}

	return nil
}

func (s *CategoryService) ValidateCategoryExists(id uint) (bool, error) {
	_, err := s.GetByID(id)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *CategoryService) GetCategoriesForSelect() ([]map[string]interface{}, error) {
	categories, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, cat := range categories {
		result = append(result, map[string]interface{}{
			"id":   cat.ID,
			"name": cat.Name,
			"slug": cat.Slug,
		})
	}

	return result, nil
}
