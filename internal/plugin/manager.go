package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"constructor-script-backend/pkg/logger"
)

var (
	// ErrManifestNotFound is returned when a plugin manifest cannot be located.
	ErrManifestNotFound = errors.New("plugin manifest not found")
)

// Metadata describes the contents of a plugin manifest file.
type Metadata struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Homepage    string `json:"homepage"`
}

// Plugin represents a plugin that is available on disk.
type Plugin struct {
	Slug     string
	Path     string
	Metadata Metadata
}

// Manager loads and caches plugin metadata from the filesystem.
type Manager struct {
	mu      sync.RWMutex
	baseDir string
	plugins map[string]*Plugin
}

// NewManager creates a new plugin manager rooted at the provided directory.
func NewManager(baseDir string) (*Manager, error) {
	cleaned := strings.TrimSpace(baseDir)
	if cleaned == "" {
		return nil, fmt.Errorf("plugin directory is required")
	}

	if err := os.MkdirAll(cleaned, 0o755); err != nil {
		return nil, fmt.Errorf("failed to ensure plugin directory: %w", err)
	}

	m := &Manager{
		baseDir: cleaned,
		plugins: make(map[string]*Plugin),
	}

	if err := m.reload(); err != nil {
		return nil, err
	}

	return m, nil
}

// BaseDir returns the root directory for plugins.
func (m *Manager) BaseDir() string {
	if m == nil {
		return ""
	}
	return m.baseDir
}

// Reload refreshes the internal plugin cache by scanning the filesystem.
func (m *Manager) Reload() error {
	if m == nil {
		return fmt.Errorf("plugin manager is not initialised")
	}
	return m.reload()
}

func (m *Manager) reload() error {
	if m == nil {
		return fmt.Errorf("plugin manager is not initialised")
	}

	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %w", err)
	}

	plugins := make(map[string]*Plugin)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(m.baseDir, entry.Name())
		manifest, err := loadManifest(dirPath)
		if err != nil {
			if errors.Is(err, ErrManifestNotFound) {
				continue
			}
			logger.Error(err, "Failed to load plugin manifest", map[string]interface{}{
				"path": dirPath,
			})
			continue
		}

		slug := strings.ToLower(strings.TrimSpace(manifest.Slug))
		if slug == "" {
			slug = strings.ToLower(strings.TrimSpace(entry.Name()))
		}
		if slug == "" {
			continue
		}

		plugins[slug] = &Plugin{
			Slug:     slug,
			Path:     dirPath,
			Metadata: manifest,
		}
	}

	m.mu.Lock()
	m.plugins = plugins
	m.mu.Unlock()

	return nil
}

// List returns all discovered plugins ordered by their slug.
func (m *Manager) List() []*Plugin {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]*Plugin, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		plugins = append(plugins, plugin)
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Slug < plugins[j].Slug
	})

	return plugins
}

// Resolve finds a plugin by slug.
func (m *Manager) Resolve(slug string) (*Plugin, bool) {
	if m == nil {
		return nil, false
	}

	cleaned := strings.ToLower(strings.TrimSpace(slug))
	if cleaned == "" {
		return nil, false
	}

	m.mu.RLock()
	plugin, ok := m.plugins[cleaned]
	m.mu.RUnlock()
	return plugin, ok
}

func loadManifest(dir string) (Metadata, error) {
	manifestPath := filepath.Join(dir, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Metadata{}, ErrManifestNotFound
		}
		return Metadata{}, fmt.Errorf("failed to read plugin manifest: %w", err)
	}

	var manifest Metadata
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Metadata{}, fmt.Errorf("failed to decode plugin manifest: %w", err)
	}

	if manifest.Slug == "" {
		manifest.Slug = manifest.Name
	}

	manifest.Slug = strings.ToLower(strings.TrimSpace(manifest.Slug))
	return manifest, nil
}
