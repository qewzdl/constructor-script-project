package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"constructor-script-backend/internal/models"

	"github.com/google/uuid"
)

// DuplicatePage creates a copy of an existing page.
func (s *PageService) DuplicatePage(pageID uint) (*models.Page, error) {
	original, err := s.pageRepo.GetByID(pageID)
	if err != nil {
		return nil, err
	}

	editor := NewSectionEditor(cloneSectionsWithFreshIDs(original.Sections), s.themes)
	sections, err := editor.Build(PrepareSectionsOptions{NormaliseSpacing: true})
	if err != nil {
		return nil, err
	}

	// Create new page with copied data
	newSlug := fmt.Sprintf("%s-copy-%d", original.Slug, time.Now().Unix())
	newTitle := fmt.Sprintf("%s (Copy)", original.Title)

	duplicate := &models.Page{
		Title:       newTitle,
		Slug:        newSlug,
		Path:        defaultPathFromSlug(newSlug),
		Sections:    sections,
		Published:   false, // New page starts as draft
		Description: original.Description,
		Content:     original.Content,
		Template:    original.Template,
		HideHeader:  original.HideHeader,
		Order:       original.Order,
		FeaturedImg: original.FeaturedImg,
	}

	if err := s.pageRepo.Create(duplicate); err != nil {
		return nil, err
	}

	return duplicate, nil
}

// ReorderSections updates the order of sections within a page.
func (s *PageService) ReorderSections(pageID uint, sectionIDs []string) (*models.Page, error) {
	page, err := s.pageRepo.GetByID(pageID)
	if err != nil {
		return nil, err
	}

	editor := NewSectionEditor(page.Sections, s.themes)
	if err := editor.Reorder(sectionIDs); err != nil {
		return nil, err
	}

	sections, err := editor.Build(PrepareSectionsOptions{NormaliseSpacing: true})
	if err != nil {
		return nil, err
	}
	page.Sections = sections

	if err := s.pageRepo.Update(page); err != nil {
		return nil, err
	}
	s.invalidatePageCaches(page)

	return page, nil
}

// AddSection adds a new section to a page.
func (s *PageService) AddSection(pageID uint, req models.AddSectionRequest) (*models.Page, error) {
	page, err := s.pageRepo.GetByID(pageID)
	if err != nil {
		return nil, err
	}

	editor := NewSectionEditor(page.Sections, s.themes)
	if err := editor.Add(req); err != nil {
		return nil, err
	}

	sections, err := editor.Build(PrepareSectionsOptions{NormaliseSpacing: true})
	if err != nil {
		return nil, err
	}
	page.Sections = sections

	if err := s.pageRepo.Update(page); err != nil {
		return nil, err
	}
	s.invalidatePageCaches(page)

	return page, nil
}

// UpdateSection updates an existing section within a page.
func (s *PageService) UpdateSection(pageID uint, sectionID string, req models.UpdateSectionRequest) (*models.Page, error) {
	page, err := s.pageRepo.GetByID(pageID)
	if err != nil {
		return nil, err
	}

	editor := NewSectionEditor(page.Sections, s.themes)
	if err := editor.Update(sectionID, req); err != nil {
		if errors.Is(err, errSectionNotFound) {
			return nil, fmt.Errorf("section not found")
		}
		return nil, err
	}

	sections, err := editor.Build(PrepareSectionsOptions{NormaliseSpacing: true})
	if err != nil {
		return nil, err
	}
	page.Sections = sections

	if err := s.pageRepo.Update(page); err != nil {
		return nil, err
	}
	s.invalidatePageCaches(page)

	return page, nil
}

// DeleteSection removes a section from a page.
func (s *PageService) DeleteSection(pageID uint, sectionID string) (*models.Page, error) {
	page, err := s.pageRepo.GetByID(pageID)
	if err != nil {
		return nil, err
	}

	editor := NewSectionEditor(page.Sections, s.themes)
	if err := editor.Delete(sectionID); err != nil {
		if errors.Is(err, errSectionNotFound) {
			return nil, fmt.Errorf("section not found")
		}
		return nil, err
	}

	sections, err := editor.Build(PrepareSectionsOptions{NormaliseSpacing: true})
	if err != nil {
		return nil, err
	}
	page.Sections = sections

	if err := s.pageRepo.Update(page); err != nil {
		return nil, err
	}
	s.invalidatePageCaches(page)

	return page, nil
}

// DuplicateSection creates a copy of an existing section within a page.
func (s *PageService) DuplicateSection(pageID uint, sectionID string) (*models.Page, error) {
	page, err := s.pageRepo.GetByID(pageID)
	if err != nil {
		return nil, err
	}

	editor := NewSectionEditor(page.Sections, s.themes)
	if err := editor.Duplicate(sectionID); err != nil {
		if errors.Is(err, errSectionNotFound) {
			return nil, fmt.Errorf("section not found")
		}
		return nil, err
	}

	sections, err := editor.Build(PrepareSectionsOptions{NormaliseSpacing: true})
	if err != nil {
		return nil, err
	}
	page.Sections = sections

	if err := s.pageRepo.Update(page); err != nil {
		return nil, err
	}
	s.invalidatePageCaches(page)

	return page, nil
}

// GetPageTemplates returns available page templates.
func (s *PageService) GetPageTemplates() []models.PageTemplate {
	return []models.PageTemplate{
		{
			ID:          "blank",
			Name:        "Blank Page",
			Description: "Start from scratch",
			Icon:        "file",
			Sections:    []models.Section{},
		},
		{
			ID:          "landing",
			Name:        "Landing Page",
			Description: "Hero section with features and CTA",
			Icon:        "layout",
			Sections: []models.Section{
				{
					ID:    uuid.New().String(),
					Type:  "standard",
					Title: "Hero Section",
					Order: 0,
				},
				{
					ID:    uuid.New().String(),
					Type:  "grid",
					Title: "Features",
					Order: 1,
				},
			},
		},
		{
			ID:          "about",
			Name:        "About Page",
			Description: "Company information and team",
			Icon:        "users",
			Sections: []models.Section{
				{
					ID:    uuid.New().String(),
					Type:  "standard",
					Title: "About Us",
					Order: 0,
				},
				{
					ID:    uuid.New().String(),
					Type:  "grid",
					Title: "Team",
					Order: 1,
				},
			},
		},
		{
			ID:          "blog",
			Name:        "Blog Page",
			Description: "Blog posts listing",
			Icon:        "book-open",
			Sections: []models.Section{
				{
					ID:    uuid.New().String(),
					Type:  "posts_list",
					Title: "Recent Posts",
					Limit: 10,
					Order: 0,
				},
				{
					ID:    uuid.New().String(),
					Type:  "categories_list",
					Title: "Categories",
					Order: 1,
				},
			},
		},
	}
}

// CreateFromTemplate creates a new page from a template.
func (s *PageService) CreateFromTemplate(templateID, title, slug string) (*models.Page, error) {
	templates := s.GetPageTemplates()

	var selectedTemplate *models.PageTemplate
	for _, tmpl := range templates {
		if tmpl.ID == templateID {
			selectedTemplate = &tmpl
			break
		}
	}

	if selectedTemplate == nil {
		return nil, fmt.Errorf("template not found")
	}

	editor := NewSectionEditor(cloneSectionsWithFreshIDs(selectedTemplate.Sections), s.themes)
	sections, err := editor.Build(PrepareSectionsOptions{NormaliseSpacing: true})
	if err != nil {
		return nil, err
	}

	page := &models.Page{
		Title:     title,
		Slug:      slug,
		Path:      defaultPathFromSlug(slug),
		Sections:  sections,
		Published: false,
	}

	if err := s.pageRepo.Create(page); err != nil {
		return nil, err
	}

	return page, nil
}

func (s *PageService) invalidatePageCaches(page *models.Page) {
	if s == nil || s.cache == nil || page == nil {
		return
	}
	s.cache.InvalidatePage(page.ID)
	s.cache.Delete("pages:all")
	if page.Path != "" {
		s.cache.Delete(fmt.Sprintf("page:path:%s", page.Path))
	}
}

// IsSlugAvailable checks if a slug is available for use.
func (s *PageService) IsSlugAvailable(slug string, excludeID *uint) (bool, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return false, fmt.Errorf("slug cannot be empty")
	}

	exists, err := s.pageRepo.ExistsBySlug(slug)
	if err != nil {
		return false, err
	}
	if !exists {
		return true, nil
	}

	// If excludeID is provided, check if it's the same page
	if excludeID != nil {
		page, err := s.pageRepo.GetBySlugAny(slug)
		if err != nil {
			return false, err
		}
		if page.ID == *excludeID {
			return true, nil
		}
	}

	return false, nil
}
