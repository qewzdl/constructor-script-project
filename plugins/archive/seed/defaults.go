package seed

import (
	"constructor-script-backend/internal/models"
	"constructor-script-backend/pkg/logger"
	archiveservice "constructor-script-backend/plugins/archive/service"
)

// EnsureDefaultStructure seeds an initial directory hierarchy when none exists.
func EnsureDefaultStructure(service *archiveservice.DirectoryService) {
	if service == nil {
		return
	}

	tree, err := service.ListTree(true)
	if err != nil {
		logger.Error(err, "Failed to inspect archive structure", nil)
		return
	}
	if len(tree) > 0 {
		return
	}

	_, err = service.Create(models.CreateArchiveDirectoryRequest{
		Name:      "Shared documents",
		Slug:      "documents",
		Published: true,
		Order:     1,
	})
	if err != nil {
		logger.Error(err, "Failed to create default archive directory", nil)
	}
}
