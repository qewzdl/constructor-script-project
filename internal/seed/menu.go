package seed

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"
)

//go:embed data/menu/*.json
var defaultMenuFS embed.FS

// EnsureDefaultMenu loads embedded menu definitions and ensures they exist in the database.
func EnsureDefaultMenu(menuService *service.MenuService) {
	if menuService == nil {
		return
	}

	entries, err := fs.ReadDir(defaultMenuFS, "data/menu")
	if err != nil {
		logger.Error(err, "Failed to read embedded menu definitions", nil)
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
		data, err := defaultMenuFS.ReadFile(fmt.Sprintf("data/menu/%s", name))
		if err != nil {
			logger.Error(err, "Failed to read embedded menu file", map[string]interface{}{"file": name})
			continue
		}

		definitions, err := parseMenuDefinitions(data)
		if err != nil {
			logger.Error(err, "Failed to parse embedded menu file", map[string]interface{}{"file": name})
			continue
		}

		for _, definition := range definitions {
			ensureMenuItem(menuService, definition, seen, name)
		}
	}
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
