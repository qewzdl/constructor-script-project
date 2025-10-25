package seed

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"
)

// EnsureDefaultMenu loads menu definitions and ensures they exist in the database.
func EnsureDefaultMenu(menuService *service.MenuService, dataFS fs.FS) {
	if menuService == nil || dataFS == nil {
		return
	}

	entries, err := fs.ReadDir(dataFS, ".")
	if err != nil {
		logger.Error(err, "Failed to read menu definitions", nil)
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	existingItems, err := menuService.List()
	if err != nil {
		logger.Error(err, "Failed to load existing menu items", nil)
		existingItems = nil
	}

	seen := make(map[string]bool, len(existingItems))
	for _, item := range existingItems {
		key := menuItemKey(item.URL, item.Location)
		if key != "" {
			seen[key] = true
		}
	}

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
			logger.Error(err, "Failed to read menu definition", map[string]interface{}{"file": name})
			continue
		}

		definitions, err := parseMenuDefinitions(data)
		if err != nil {
			logger.Error(err, "Failed to parse menu definition", map[string]interface{}{"file": name})
			continue
		}

		for _, definition := range definitions {
			ensureMenuItem(menuService, definition, seen, name)
		}
	}
}

func ResetMenu(menuService *service.MenuService, dataFS fs.FS) error {
	if menuService == nil || dataFS == nil {
		return nil
	}

	entries, err := fs.ReadDir(dataFS, ".")
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var (
		definitions []models.CreateMenuItemRequest
		errs        []error
	)

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
			logger.Error(readErr, "Failed to read menu definition", map[string]interface{}{"file": name})
			errs = append(errs, fmt.Errorf("read menu definition %s: %w", name, readErr))
			continue
		}

		parsed, parseErr := parseMenuDefinitions(data)
		if parseErr != nil {
			logger.Error(parseErr, "Failed to parse menu definition", map[string]interface{}{"file": name})
			errs = append(errs, fmt.Errorf("parse menu definition %s: %w", name, parseErr))
			continue
		}

		definitions = append(definitions, parsed...)
	}

	if err := menuService.DeleteAll(); err != nil {
		logger.Error(err, "Failed to clear existing menu items", nil)
		errs = append(errs, fmt.Errorf("clear existing menu items: %w", err))
	} else {
		for _, definition := range definitions {
			if _, err := menuService.Create(definition); err != nil {
				logger.Error(err, "Failed to recreate default menu item", map[string]interface{}{"url": definition.URL, "location": definition.Location})
				errs = append(errs, fmt.Errorf("create menu item %s@%s: %w", definition.URL, definition.Location, err))
				continue
			}
			logger.Info("Reset default menu item", map[string]interface{}{"url": definition.URL, "location": definition.Location})
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func ensureMenuItem(menuService *service.MenuService, definition models.CreateMenuItemRequest, seen map[string]bool, source string) {
	title := strings.TrimSpace(definition.Title)
	url := strings.TrimSpace(definition.URL)
	if title == "" || url == "" {
		return
	}

	location := normalizeMenuLocation(definition.Location)

	key := menuItemKey(url, location)
	if key == "" {
		return
	}
	if seen[key] {
		logger.Info("Default menu item already present", map[string]interface{}{"url": url, "location": location, "source": source})
		return
	}

	definition.Title = title
	definition.URL = url
	definition.Location = location

	if _, err := menuService.Create(definition); err != nil {
		logger.Error(err, "Failed to create default menu item", map[string]interface{}{"url": url, "location": location, "source": source})
		return
	}

	seen[key] = true
	logger.Info("Ensured default menu item", map[string]interface{}{"url": url, "location": location, "source": source})
}

func parseMenuDefinitions(data []byte) ([]models.CreateMenuItemRequest, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}

	if trimmed[0] == '[' {
		var definitions []models.CreateMenuItemRequest
		if err := json.Unmarshal(trimmed, &definitions); err != nil {
			return nil, err
		}
		return definitions, nil
	}

	var definition models.CreateMenuItemRequest
	if err := json.Unmarshal(trimmed, &definition); err != nil {
		return nil, err
	}

	return []models.CreateMenuItemRequest{definition}, nil
}

func normalizeMenuLocation(location string) string {
	cleaned := strings.ToLower(strings.TrimSpace(location))
	if cleaned == "" {
		return "header"
	}
	return cleaned
}

func menuItemKey(url, location string) string {
	cleanedURL := strings.TrimSpace(strings.ToLower(url))
	cleanedLocation := strings.TrimSpace(strings.ToLower(location))
	if cleanedURL == "" {
		return ""
	}
	if cleanedLocation == "" {
		cleanedLocation = "header"
	}
	return cleanedLocation + "|" + cleanedURL
}
