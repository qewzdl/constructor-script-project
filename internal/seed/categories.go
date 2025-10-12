package seed

import (
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"
)

func EnsureDefaultCategory(categoryService *service.CategoryService) {
	if categoryService == nil {
		return
	}

	category, created, err := categoryService.EnsureDefaultCategory()
	if err != nil {
		logger.Error(err, "Failed to ensure default category", nil)
		return
	}

	fields := map[string]interface{}{
		"id":   category.ID,
		"name": category.Name,
		"slug": category.Slug,
	}

	if created {
		logger.Info("Created default category", fields)
	} else {
		logger.Info("Default category already present", fields)
	}
}
