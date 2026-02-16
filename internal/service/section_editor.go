package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/theme"

	"github.com/google/uuid"
)

var errSectionNotFound = errors.New("section not found")

// SectionEditor centralises mutations over section collections and keeps IDs/order stable.
type SectionEditor struct {
	themes    *theme.Manager
	sections  []models.Section
	nextOrder int
}

func NewSectionEditor(sections []models.Section, manager *theme.Manager) *SectionEditor {
	editor := &SectionEditor{
		themes:   manager,
		sections: cloneSections(sections),
	}
	editor.reindex()
	return editor
}

func (e *SectionEditor) Sections() []models.Section {
	if e == nil {
		return nil
	}
	return cloneSections(e.sections)
}

func (e *SectionEditor) Build(opts PrepareSectionsOptions) (models.PostSections, error) {
	if e == nil {
		return models.PostSections{}, errors.New("section editor is nil")
	}
	return PrepareSections(e.sections, e.themes, opts)
}

func (e *SectionEditor) Add(req models.AddSectionRequest) error {
	if e == nil {
		return errors.New("section editor is nil")
	}

	animation := constants.NormaliseSectionAnimation(req.Animation)
	animationBlur := constants.DefaultSectionAnimationBlur
	if req.AnimationBlur != nil {
		animationBlur = constants.NormaliseSectionAnimationBlur(req.AnimationBlur)
	}

	section := models.Section{
		ID:              uuid.New().String(),
		Type:            strings.TrimSpace(req.Type),
		Variation:       strings.TrimSpace(req.Variation),
		Title:           req.Title,
		Description:     req.Description,
		Image:           strings.TrimSpace(req.Image),
		Mode:            strings.TrimSpace(req.Mode),
		Elements:        []models.SectionElement{},
		Settings:        cloneSettings(req.Settings),
		PaddingVertical: copyIntPointer(req.PaddingVertical),
		MarginVertical:  copyIntPointer(req.MarginVertical),
		StyleGridItems:  cloneBoolPointer(req.StyleGridItems),
		Animation:       animation,
		AnimationBlur:   boolPtr(animationBlur),
	}

	if req.Limit != nil {
		section.Limit = *req.Limit
	}
	if req.Disabled != nil {
		section.Disabled = *req.Disabled
	}

	e.sections = append(e.sections, section)
	e.reindex()
	return nil
}

func (e *SectionEditor) Update(sectionID string, req models.UpdateSectionRequest) error {
	if e == nil {
		return errors.New("section editor is nil")
	}

	index := e.findIndex(sectionID)
	if index < 0 {
		return errSectionNotFound
	}

	section := e.sections[index]

	if req.Type != nil {
		section.Type = strings.TrimSpace(*req.Type)
	}
	if req.Variation != nil {
		section.Variation = strings.TrimSpace(*req.Variation)
	}
	if req.Title != nil {
		section.Title = *req.Title
	}
	if req.Description != nil {
		section.Description = *req.Description
	}
	if req.Image != nil {
		section.Image = strings.TrimSpace(*req.Image)
	}
	if req.Elements != nil {
		section.Elements = cloneSectionElements(*req.Elements)
	}
	if req.PaddingVertical != nil {
		section.PaddingVertical = intPtr(*req.PaddingVertical)
	}
	if req.MarginVertical != nil {
		section.MarginVertical = intPtr(*req.MarginVertical)
	}
	if req.Limit != nil {
		section.Limit = *req.Limit
	}
	if req.Mode != nil {
		section.Mode = strings.TrimSpace(*req.Mode)
	}
	if req.StyleGridItems != nil {
		section.StyleGridItems = boolPtr(*req.StyleGridItems)
	}
	if req.Disabled != nil {
		section.Disabled = *req.Disabled
	}
	if req.Settings != nil {
		section.Settings = cloneSettings(*req.Settings)
	}
	if req.Animation != nil {
		section.Animation = constants.NormaliseSectionAnimation(*req.Animation)
	}
	if req.AnimationBlur != nil {
		blur := constants.NormaliseSectionAnimationBlur(req.AnimationBlur)
		section.AnimationBlur = boolPtr(blur)
	}

	e.sections[index] = section
	e.reindex()
	return nil
}

func (e *SectionEditor) Delete(sectionID string) error {
	if e == nil {
		return errors.New("section editor is nil")
	}

	index := e.findIndex(sectionID)
	if index < 0 {
		return errSectionNotFound
	}

	e.sections = append(e.sections[:index], e.sections[index+1:]...)
	e.reindex()
	return nil
}

func (e *SectionEditor) Duplicate(sectionID string) error {
	if e == nil {
		return errors.New("section editor is nil")
	}

	index := e.findIndex(sectionID)
	if index < 0 {
		return errSectionNotFound
	}

	duplicate := cloneSection(e.sections[index])
	duplicate.ID = uuid.New().String()
	duplicate.Title = duplicateTitle(duplicate.Title)
	for i := range duplicate.Elements {
		duplicate.Elements[i].ID = uuid.New().String()
	}

	insertAt := index + 1
	reordered := make([]models.Section, 0, len(e.sections)+1)
	reordered = append(reordered, e.sections[:insertAt]...)
	reordered = append(reordered, duplicate)
	reordered = append(reordered, e.sections[insertAt:]...)

	e.sections = reordered
	e.reindex()
	return nil
}

func (e *SectionEditor) Reorder(sectionIDs []string) error {
	if e == nil {
		return errors.New("section editor is nil")
	}

	if len(sectionIDs) != len(e.sections) {
		return fmt.Errorf("expected %d section ids, got %d", len(e.sections), len(sectionIDs))
	}

	byID := make(map[string]models.Section, len(e.sections))
	for _, section := range e.sections {
		id := strings.TrimSpace(section.ID)
		if id == "" {
			return errors.New("cannot reorder sections with empty id")
		}
		if _, exists := byID[id]; exists {
			return fmt.Errorf("duplicate section id '%s' in current state", id)
		}
		byID[id] = section
	}

	seen := make(map[string]struct{}, len(sectionIDs))
	reordered := make([]models.Section, 0, len(sectionIDs))
	for idx, rawID := range sectionIDs {
		id := strings.TrimSpace(rawID)
		if id == "" {
			return fmt.Errorf("section id at position %d is empty", idx)
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("section id '%s' is duplicated in reorder payload", id)
		}
		section, ok := byID[id]
		if !ok {
			return fmt.Errorf("section id '%s' does not exist", id)
		}
		seen[id] = struct{}{}
		reordered = append(reordered, section)
	}

	e.sections = reordered
	e.reindex()
	return nil
}

func (e *SectionEditor) findIndex(sectionID string) int {
	id := strings.TrimSpace(sectionID)
	if id == "" {
		return -1
	}

	for i := range e.sections {
		if e.sections[i].ID == id {
			return i
		}
	}
	return -1
}

func (e *SectionEditor) reindex() {
	e.nextOrder = 1
	for i := range e.sections {
		section := &e.sections[i]
		if strings.TrimSpace(section.ID) == "" {
			section.ID = uuid.New().String()
		}
		section.Order = e.nextOrder
		e.nextOrder++

		for j := range section.Elements {
			element := &section.Elements[j]
			if strings.TrimSpace(element.ID) == "" {
				element.ID = uuid.New().String()
			}
			element.Order = j + 1
		}
	}
}

func cloneSections(source []models.Section) []models.Section {
	if len(source) == 0 {
		return []models.Section{}
	}

	result := make([]models.Section, len(source))
	for i := range source {
		result[i] = cloneSection(source[i])
	}
	return result
}

func cloneSection(source models.Section) models.Section {
	result := source
	result.Settings = cloneSettings(source.Settings)
	result.Elements = cloneSectionElements(source.Elements)
	result.PaddingVertical = copyIntPointer(source.PaddingVertical)
	result.MarginVertical = copyIntPointer(source.MarginVertical)
	result.StyleGridItems = cloneBoolPointer(source.StyleGridItems)
	result.AnimationBlur = cloneBoolPointer(source.AnimationBlur)
	return result
}

func cloneSectionElements(source []models.SectionElement) []models.SectionElement {
	if len(source) == 0 {
		return nil
	}

	result := make([]models.SectionElement, len(source))
	for i := range source {
		element := source[i]
		element.Content = cloneAny(element.Content)
		result[i] = element
	}
	return result
}

func cloneSettings(source map[string]interface{}) map[string]interface{} {
	if len(source) == 0 {
		return nil
	}

	result := make(map[string]interface{}, len(source))
	for key, value := range source {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		result[trimmed] = cloneAny(value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func cloneAny(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return value
	}

	var result interface{}
	if err := json.Unmarshal(payload, &result); err != nil {
		return value
	}
	return result
}

func copyIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func cloneBoolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func boolPtr(value bool) *bool {
	return &value
}

func duplicateTitle(original string) string {
	title := strings.TrimSpace(original)
	if title == "" {
		return "Section (Copy)"
	}
	return fmt.Sprintf("%s (Copy)", title)
}

func cloneSectionsWithFreshIDs(source []models.Section) []models.Section {
	editor := NewSectionEditor(source, nil)
	for i := range editor.sections {
		editor.sections[i].ID = uuid.New().String()
		for j := range editor.sections[i].Elements {
			editor.sections[i].Elements[j].ID = uuid.New().String()
		}
	}
	editor.reindex()
	return editor.Sections()
}
