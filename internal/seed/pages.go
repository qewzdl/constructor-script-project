package seed

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/utils"
)

// EnsureDefaultPages loads page definitions from the provided filesystem and ensures they exist in the database.
func EnsureDefaultPages(pageService *service.PageService, dataFS fs.FS) {
	if pageService == nil || dataFS == nil {
		return
	}

	entries, err := fs.ReadDir(dataFS, ".")
	if err != nil {
		logger.Error(err, "Failed to read page definitions", nil)
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
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}
		data, err := fs.ReadFile(dataFS, name)
		if err != nil {
			logger.Error(err, "Failed to read page definition", map[string]interface{}{"file": name})
			continue
		}

		definitions, err := parsePageDefinitions(data)
		if err != nil {
			logger.Error(err, "Failed to parse page definition", map[string]interface{}{"file": name})
			continue
		}

		for _, definition := range definitions {
			ensurePage(pageService, definition, name)
		}
	}
}

func ResetPages(pageService *service.PageService, dataFS fs.FS) error {
	if pageService == nil || dataFS == nil {
		return nil
	}

	entries, err := fs.ReadDir(dataFS, ".")
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var errs []error

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}
		data, readErr := fs.ReadFile(dataFS, name)
		if readErr != nil {
			logger.Error(readErr, "Failed to read page definition", map[string]interface{}{"file": name})
			errs = append(errs, fmt.Errorf("read page definition %s: %w", name, readErr))
			continue
		}

		definitions, parseErr := parsePageDefinitions(data)
		if parseErr != nil {
			logger.Error(parseErr, "Failed to parse page definition", map[string]interface{}{"file": name})
			errs = append(errs, fmt.Errorf("parse page definition %s: %w", name, parseErr))
			continue
		}

		for _, definition := range definitions {
			if err := resetPage(pageService, definition, name); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func ensurePage(pageService *service.PageService, definition models.CreatePageRequest, source string) {
	slug := definition.Slug
	if slug == "" {
		slug = utils.GenerateSlug(definition.Title)
	} else {
		slug = utils.GenerateSlug(slug)
	}

	definition.Slug = slug

	pathValue := strings.TrimSpace(definition.Path)
	if pathValue == "" {
		if slug == "home" {
			pathValue = "/"
		} else {
			pathValue = "/" + slug
		}
	}
	definition.Path = pathValue

	if pathValue != "" {
		if _, err := pageService.GetByPath(pathValue); err == nil {
			logger.Info("Default page already present", map[string]interface{}{"path": pathValue, "source": source})
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error(err, "Failed to verify default page", map[string]interface{}{"path": pathValue, "source": source})
			return
		}
	}

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

	logger.Info("Ensured default page", map[string]interface{}{"slug": slug, "path": pathValue, "source": source})
}

func resetPage(pageService *service.PageService, definition models.CreatePageRequest, source string) error {
	if pageService == nil {
		return errors.New("page service not configured")
	}

	slug := definition.Slug
	if slug == "" {
		slug = definition.Title
	}
	slug = utils.GenerateSlug(slug)

	page, err := pageService.ApplyDefinition(definition)
	if err != nil {
		logger.Error(err, "Failed to reset default page", map[string]interface{}{"slug": slug, "source": source})
		return fmt.Errorf("reset page %q failed: %w", slug, err)
	}

	if page != nil {
		slug = page.Slug
	}

	logger.Info("Reset default page", map[string]interface{}{"slug": slug, "source": source})
	return nil
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
