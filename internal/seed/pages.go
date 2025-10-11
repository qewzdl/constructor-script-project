package seed

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/utils"
)

//go:embed data/pages/*.json
var defaultPagesFS embed.FS

// EnsureDefaultPages loads embedded page definitions and makes sure they exist in the database.
func EnsureDefaultPages(pageService *service.PageService) {
	entries, err := fs.ReadDir(defaultPagesFS, "data/pages")
	if err != nil {
		logger.Error(err, "Failed to read embedded page definitions", nil)
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		data, err := defaultPagesFS.ReadFile(fmt.Sprintf("data/pages/%s", name))
		if err != nil {
			logger.Error(err, "Failed to read embedded page file", map[string]interface{}{"file": name})
			continue
		}

		definitions, err := parsePageDefinitions(data)
		if err != nil {
			logger.Error(err, "Failed to parse embedded page file", map[string]interface{}{"file": name})
			continue
		}

		for _, definition := range definitions {
			ensurePage(pageService, definition, name)
		}
	}
}

func ensurePage(pageService *service.PageService, definition models.CreatePageRequest, source string) {
	slug := definition.Slug
	if slug == "" {
		slug = utils.GenerateSlug(definition.Title)
	} else {
		slug = utils.GenerateSlug(slug)
	}

	definition.Slug = slug

	if _, err := pageService.GetBySlug(slug); err == nil {
		logger.Info("Default page already present", map[string]interface{}{"slug": slug, "source": source})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error(err, "Failed to verify default page", map[string]interface{}{"slug": slug, "source": source})
		return
	}

	if _, err := pageService.Create(definition); err != nil {
		logger.Error(err, "Failed to create default page", map[string]interface{}{"slug": slug, "source": source})
		return
	}

	logger.Info("Ensured default page", map[string]interface{}{"slug": slug, "source": source})
}

func parsePageDefinitions(data []byte) ([]models.CreatePageRequest, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}

	if trimmed[0] == '[' {
		var definitions []models.CreatePageRequest
		if err := json.Unmarshal(trimmed, &definitions); err != nil {
			return nil, err
		}
		return definitions, nil
	}

	var definition models.CreatePageRequest
	if err := json.Unmarshal(trimmed, &definition); err != nil {
		return nil, err
	}

	return []models.CreatePageRequest{definition}, nil
}
