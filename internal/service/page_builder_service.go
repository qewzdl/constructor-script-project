package service

import (
	"fmt"
	"strings"
	"time"

	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"

	"github.com/google/uuid"
)

// DuplicatePage creates a copy of an existing page.
func (s *PageService) DuplicatePage(pageID uint) (*models.Page, error) {
	original, err := s.GetByID(pageID)
	if err != nil {
		return nil, err
	}

	// Create new page with copied data
	newSlug := fmt.Sprintf("%s-copy-%d", original.Slug, time.Now().Unix())
	newTitle := fmt.Sprintf("%s (Copy)", original.Title)

	duplicate := &models.Page{
		Title:       newTitle,
		Slug:        newSlug,
		Sections:    original.Sections, // Deep copy sections
		Published:   false,             // New page starts as draft
		Description: original.Description,
	}

	// Generate new IDs for sections
	for i := range duplicate.Sections {
		duplicate.Sections[i].ID = uuid.New().String()
	}

	if err := s.pageRepo.Create(duplicate); err != nil {
		return nil, err
	}

	return duplicate, nil
}

// ReorderSections updates the order of sections within a page.
func (s *PageService) ReorderSections(pageID uint, sectionIDs []string) (*models.Page, error) {
	page, err := s.GetByID(pageID)
	if err != nil {
		return nil, err
	}

	// Create a map for quick lookup
	sectionMap := make(map[string]models.Section)
	for _, section := range page.Sections {
		sectionMap[section.ID] = section
	}

	// Reorder sections based on the provided IDs
	newSections := make([]models.Section, 0, len(sectionIDs))
	for i, id := range sectionIDs {
		if section, ok := sectionMap[id]; ok {
			section.Order = i
			newSections = append(newSections, section)
		}
	}

	page.Sections = newSections

	if err := s.pageRepo.Update(page); err != nil {
		return nil, err
	}

	return page, nil
}

// AddSection adds a new section to a page.
func (s *PageService) AddSection(pageID uint, req models.AddSectionRequest) (*models.Page, error) {
	page, err := s.GetByID(pageID)
	if err != nil {
		return nil, err
	}

	newSection := models.Section{
		ID:          uuid.New().String(),
		Type:        req.Type,
		Title:       req.Title,
		Description: req.Description,
		Order:       len(page.Sections),
		Elements:    make([]models.SectionElement, 0),
		Animation:   constants.NormaliseSectionAnimation(req.Animation),
	}
	blurEnabled := constants.DefaultSectionAnimationBlur
	if req.AnimationBlur != nil {
		blurEnabled = *req.AnimationBlur
	}
	newSection.AnimationBlur = &blurEnabled

	if req.Disabled != nil {
		newSection.Disabled = *req.Disabled
	}

	if req.PaddingVertical != nil {
		newSection.PaddingVertical = req.PaddingVertical
	}
	if req.MarginVertical != nil {
		newSection.MarginVertical = req.MarginVertical
	}

	page.Sections = append(page.Sections, newSection)

	if err := s.pageRepo.Update(page); err != nil {
		return nil, err
	}

	return page, nil
}

// UpdateSection updates an existing section within a page.
func (s *PageService) UpdateSection(pageID uint, sectionID string, req models.UpdateSectionRequest) (*models.Page, error) {
	page, err := s.GetByID(pageID)
	if err != nil {
		return nil, err
	}

	found := false
	for i := range page.Sections {
		if page.Sections[i].ID == sectionID {
			if req.Title != nil {
				page.Sections[i].Title = *req.Title
			}
			if req.Description != nil {
				page.Sections[i].Description = *req.Description
			}
			if req.Type != nil {
				page.Sections[i].Type = *req.Type
			}
			if req.Elements != nil {
				page.Sections[i].Elements = *req.Elements
			}
			if req.PaddingVertical != nil {
				page.Sections[i].PaddingVertical = req.PaddingVertical
			}
			if req.MarginVertical != nil {
				page.Sections[i].MarginVertical = req.MarginVertical
			}
			if req.Limit != nil {
				page.Sections[i].Limit = *req.Limit
			}
			if req.Mode != nil {
				page.Sections[i].Mode = *req.Mode
			}
			if req.StyleGridItems != nil {
				page.Sections[i].StyleGridItems = req.StyleGridItems
			}
			if req.Disabled != nil {
				page.Sections[i].Disabled = *req.Disabled
			}
			if req.Animation != nil {
				page.Sections[i].Animation = constants.NormaliseSectionAnimation(*req.Animation)
			}
			if req.AnimationBlur != nil {
				blur := constants.NormaliseSectionAnimationBlur(req.AnimationBlur)
				page.Sections[i].AnimationBlur = &blur
			}
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("section not found")
	}

	if err := s.pageRepo.Update(page); err != nil {
		return nil, err
	}

	return page, nil
}

// DeleteSection removes a section from a page.
func (s *PageService) DeleteSection(pageID uint, sectionID string) (*models.Page, error) {
	page, err := s.GetByID(pageID)
	if err != nil {
		return nil, err
	}

	newSections := make([]models.Section, 0, len(page.Sections)-1)
	for _, section := range page.Sections {
		if section.ID != sectionID {
			newSections = append(newSections, section)
		}
	}

	// Reorder after deletion
	for i := range newSections {
		newSections[i].Order = i
	}

	page.Sections = newSections

	if err := s.pageRepo.Update(page); err != nil {
		return nil, err
	}

	return page, nil
}

// DuplicateSection creates a copy of an existing section within a page.
func (s *PageService) DuplicateSection(pageID uint, sectionID string) (*models.Page, error) {
	page, err := s.GetByID(pageID)
	if err != nil {
		return nil, err
	}

	var originalSection *models.Section
	var insertIndex int

	for i := range page.Sections {
		if page.Sections[i].ID == sectionID {
			originalSection = &page.Sections[i]
			insertIndex = i + 1
			break
		}
	}

	if originalSection == nil {
		return nil, fmt.Errorf("section not found")
	}

	// Create duplicate with new ID
	duplicate := *originalSection
	duplicate.ID = uuid.New().String()
	duplicate.Title = fmt.Sprintf("%s (Copy)", originalSection.Title)

	// Generate new IDs for elements
	for i := range duplicate.Elements {
		duplicate.Elements[i].ID = uuid.New().String()
	}

	// Insert duplicate after original
	newSections := make([]models.Section, 0, len(page.Sections)+1)
	newSections = append(newSections, page.Sections[:insertIndex]...)
	newSections = append(newSections, duplicate)
	newSections = append(newSections, page.Sections[insertIndex:]...)

	// Reorder
	for i := range newSections {
		newSections[i].Order = i
	}

	page.Sections = newSections

	if err := s.pageRepo.Update(page); err != nil {
		return nil, err
	}

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

	page := &models.Page{
		Title:     title,
		Slug:      slug,
		Sections:  selectedTemplate.Sections,
		Published: false,
	}

	// Generate new IDs for all sections
	for i := range page.Sections {
		page.Sections[i].ID = uuid.New().String()
	}

	if err := s.pageRepo.Create(page); err != nil {
		return nil, err
	}

	return page, nil
}

// IsSlugAvailable checks if a slug is available for use.
func (s *PageService) IsSlugAvailable(slug string, excludeID *uint) (bool, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return false, fmt.Errorf("slug cannot be empty")
	}

	page, err := s.pageRepo.GetBySlug(slug)
	if err != nil {
		// Slug is available if page doesn't exist
		return true, nil
	}

	// If excludeID is provided, check if it's the same page
	if excludeID != nil && page.ID == *excludeID {
		return true, nil
	}

	return false, nil
}
