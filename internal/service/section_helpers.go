package service

import (
	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/theme"
)

func sectionDefinitionsFromManager(manager *theme.Manager) map[string]theme.SectionDefinition {
	if manager == nil {
		return theme.DefaultSectionDefinitions()
	}
	if active := manager.Active(); active != nil {
		defs := active.SectionDefinitions()
		if len(defs) > 0 {
			return defs
		}
	}
	return theme.DefaultSectionDefinitions()
}

func elementDefinitionsFromManager(manager *theme.Manager) map[string]theme.ElementDefinition {
	if manager == nil {
		return theme.DefaultElementDefinitions()
	}
	if active := manager.Active(); active != nil {
		defs := active.ElementDefinitions()
		if len(defs) > 0 {
			return defs
		}
	}
	return theme.DefaultElementDefinitions()
}

func clampSectionLimit(value int, setting theme.SectionSettingDefinition) int {
	result := value
	if result <= 0 {
		if setting.Default != nil {
			result = *setting.Default
		} else {
			result = constants.DefaultPostListSectionLimit
		}
	}
	if setting.Min != nil && result < *setting.Min {
		result = *setting.Min
	}
	if setting.Max != nil && result > *setting.Max {
		result = *setting.Max
	} else if setting.Max == nil && result > constants.MaxPostListSectionLimit {
		result = constants.MaxPostListSectionLimit
	}
	if result <= 0 {
		result = 1
	}
	return result
}

func intPtr(value int) *int {
	return &value
}
