package service

import (
	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
)

// GetPageBuilderConfig returns configuration for the page builder UI.
func (s *PageService) GetPageBuilderConfig() models.PageBuilderConfig {
	return s.GetPageBuilderConfigWithOptions(SectionCatalogOptions{
		BlogEnabled:    true,
		CoursesEnabled: true,
	})
}

// GetPageBuilderConfigWithOptions returns configuration for the page builder UI
// using feature flags that control which section families are exposed.
func (s *PageService) GetPageBuilderConfigWithOptions(options SectionCatalogOptions) models.PageBuilderConfig {
	animationOptions := constants.SectionAnimationOptions()
	animations := make([]models.SectionAnimationOption, 0, len(animationOptions))
	for _, option := range animationOptions {
		animations = append(animations, models.SectionAnimationOption{
			Value:       option.Value,
			Label:       option.Label,
			Description: option.Description,
		})
	}

	defaultPadding := constants.DefaultSectionPadding
	if s != nil && s.themes != nil {
		if active := s.themes.Active(); active != nil {
			defaultPadding = active.DefaultSectionPadding()
		}
	}

	catalog := s.SectionCatalog(options)

	return models.PageBuilderConfig{
		AvailableSections:    catalog.SectionTypeConfigs(),
		DefaultPadding:       defaultPadding,
		DefaultMargin:        constants.DefaultSectionMargin,
		PaddingOptions:       constants.SectionPaddingOptions(),
		MarginOptions:        constants.SectionMarginOptions(),
		SectionAnimations:    animations,
		DefaultAnimation:     constants.DefaultSectionAnimation,
		DefaultAnimationBlur: constants.DefaultSectionAnimationBlur,
	}
}
