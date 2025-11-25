package service

import (
	"errors"
	"strings"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/internal/theme"
)

type ContentService struct {
	contentRepo repository.CourseContentRepository
	themes      *theme.Manager
}

func NewContentService(contentRepo repository.CourseContentRepository, themes *theme.Manager) *ContentService {
	return &ContentService{
		contentRepo: contentRepo,
		themes:      themes,
	}
}

func (s *ContentService) SetThemeManager(manager *theme.Manager) {
	if s == nil {
		return
	}
	s.themes = manager
}

func (s *ContentService) Create(req models.CreateCourseContentRequest) (*models.CourseContent, error) {
	if s == nil || s.contentRepo == nil {
		return nil, errors.New("course content repository is not configured")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, newValidationError("content title is required")
	}

	sections, err := service.PrepareSections(req.Sections, s.themes, service.PrepareSectionsOptions{NormaliseSpacing: true})
	if err != nil {
		return nil, err
	}

	content := models.CourseContent{
		Title:       title,
		Description: strings.TrimSpace(req.Description),
		Sections:    sections,
	}

	if err := s.contentRepo.Create(&content); err != nil {
		return nil, err
	}

	return &content, nil
}

func (s *ContentService) Update(id uint, req models.UpdateCourseContentRequest) (*models.CourseContent, error) {
	if s == nil || s.contentRepo == nil {
		return nil, errors.New("course content repository is not configured")
	}

	content, err := s.contentRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return nil, newValidationError("content title is required")
		}
		content.Title = title
	}

	if req.Description != nil {
		content.Description = strings.TrimSpace(*req.Description)
	}

	if req.Sections != nil {
		sections, err := service.PrepareSections(*req.Sections, s.themes, service.PrepareSectionsOptions{NormaliseSpacing: true})
		if err != nil {
			return nil, err
		}
		content.Sections = sections
	}

	if err := s.contentRepo.Update(content); err != nil {
		return nil, err
	}

	return content, nil
}

func (s *ContentService) Delete(id uint) error {
	if s == nil || s.contentRepo == nil {
		return errors.New("course content repository is not configured")
	}
	return s.contentRepo.Delete(id)
}

func (s *ContentService) GetByID(id uint) (*models.CourseContent, error) {
	if s == nil || s.contentRepo == nil {
		return nil, errors.New("course content repository is not configured")
	}
	return s.contentRepo.GetByID(id)
}

func (s *ContentService) List() ([]models.CourseContent, error) {
	if s == nil || s.contentRepo == nil {
		return nil, errors.New("course content repository is not configured")
	}

	contents, err := s.contentRepo.List()
	if err != nil {
		return nil, err
	}

	for i := range contents {
		contents[i].Sections = service.NormaliseSections(contents[i].Sections)
	}

	return contents, nil
}
