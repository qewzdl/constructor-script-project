package theme

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

type Metadata struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Version        string `json:"version"`
	Author         string `json:"author"`
	PreviewImage   string `json:"preview_image"`
	DefaultLogo    string `json:"default_logo"`
	DefaultFavicon string `json:"default_favicon"`
}

type Theme struct {
	Slug         string
	Path         string
	TemplatesDir string
	StaticDir    string
	DataDir      string
	Metadata     Metadata
}

type Manager struct {
	baseDir string

	mu     sync.RWMutex
	themes map[string]*Theme
	active *Theme
}

func NewManager(baseDir string) (*Manager, error) {
	cleaned := filepath.Clean(strings.TrimSpace(baseDir))
	if cleaned == "" {
		return nil, errors.New("themes directory is required")
	}

	info, err := os.Stat(cleaned)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("themes path must be a directory")
	}

	entries, err := os.ReadDir(cleaned)
	if err != nil {
		return nil, err
	}

	themes := make(map[string]*Theme)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		slug := entry.Name()
		themePath := filepath.Join(cleaned, slug)
		theme, loadErr := loadTheme(themePath, slug)
		if loadErr != nil {
			return nil, loadErr
		}
		themes[theme.Slug] = theme
	}

	if len(themes) == 0 {
		return nil, errors.New("no themes found")
	}

	return &Manager{baseDir: cleaned, themes: themes}, nil
}

func (m *Manager) List() []*Theme {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*Theme, 0, len(m.themes))
	for _, theme := range m.themes {
		list = append(list, theme)
	}

	sort.Slice(list, func(i, j int) bool {
		left := strings.ToLower(list[i].Metadata.Name)
		right := strings.ToLower(list[j].Metadata.Name)
		if left == right {
			return list[i].Slug < list[j].Slug
		}
		return left < right
	})

	return list
}

func (m *Manager) Active() *Theme {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

func (m *Manager) Activate(slug string) error {
	cleaned := strings.ToLower(strings.TrimSpace(slug))
	if cleaned == "" {
		return errors.New("theme slug is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	theme, ok := m.themes[cleaned]
	if !ok {
		return errors.New("theme not found: " + cleaned)
	}

	m.active = theme
	return nil
}

func (m *Manager) Resolve(slug string) (*Theme, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	theme, ok := m.themes[strings.ToLower(strings.TrimSpace(slug))]
	return theme, ok
}

func (m *Manager) AssetModTime(path string) (time.Time, error) {
	m.mu.RLock()
	theme := m.active
	m.mu.RUnlock()

	if theme == nil {
		return time.Time{}, errors.New("no active theme")
	}

	return theme.AssetModTime(path)
}

func loadTheme(themePath, slug string) (*Theme, error) {
	info, err := os.Stat(themePath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("invalid theme directory: " + themePath)
	}

	slugValue := strings.ToLower(strings.TrimSpace(slug))
	if slugValue == "" {
		slugValue = slug
	}

	metadata, err := readMetadata(themePath)
	if err != nil {
		return nil, err
	}

	if metadata.Name == "" {
		metadata.Name = humanizeSlug(slugValue)
	}

	theme := &Theme{
		Slug:         slugValue,
		Path:         themePath,
		TemplatesDir: filepath.Join(themePath, "templates"),
		StaticDir:    filepath.Join(themePath, "static"),
		DataDir:      filepath.Join(themePath, "data"),
		Metadata:     metadata,
	}

	if _, err := os.Stat(theme.TemplatesDir); err != nil {
		return nil, errors.New("theme missing templates directory: " + slug)
	}

	if _, err := os.Stat(theme.StaticDir); err != nil {
		return nil, errors.New("theme missing static directory: " + slug)
	}

	return theme, nil
}

func readMetadata(themePath string) (Metadata, error) {
	filePath := filepath.Join(themePath, "theme.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Metadata{}, nil
		}
		return Metadata{}, err
	}

	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return Metadata{}, err
	}

	return metadata, nil
}

func (t *Theme) PagesFS() fs.FS {
	return t.dataFS("pages")
}

func (t *Theme) MenuFS() fs.FS {
	return t.dataFS("menu")
}

func (t *Theme) PostsFS() fs.FS {
	return t.dataFS("posts")
}

func (t *Theme) dataFS(dir string) fs.FS {
	if dir == "" {
		return nil
	}

	path := filepath.Join(t.DataDir, dir)
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		return nil
	}

	return os.DirFS(path)
}

func (t *Theme) AssetModTime(path string) (time.Time, error) {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		return time.Time{}, errors.New("asset path is required")
	}

	trimmed := strings.TrimPrefix(cleaned, "./")
	trimmed = strings.TrimPrefix(trimmed, "/")

	if !strings.HasPrefix(trimmed, "static/") {
		return time.Time{}, os.ErrNotExist
	}

	relative := strings.TrimPrefix(trimmed, "static/")
	full := filepath.Join(t.StaticDir, filepath.FromSlash(relative))
	info, err := os.Stat(full)
	if err != nil {
		return time.Time{}, err
	}

	return info.ModTime(), nil
}

func (t *Theme) TemplateNames() ([]string, error) {
	entries, err := os.ReadDir(t.TemplatesDir)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		names = append(names, entry.Name())
	}

	sort.Strings(names)
	return names, nil
}

func (t *Theme) TemplatesPath() string {
	return t.TemplatesDir
}

func (t *Theme) StaticPath() string {
	return t.StaticDir
}

func (t *Theme) DataPath() string {
	return t.DataDir
}

func (t *Theme) MetadataOrDefault() Metadata {
	return t.Metadata
}

func humanizeSlug(value string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return "Theme"
	}

	parts := strings.FieldsFunc(cleaned, func(r rune) bool {
		switch r {
		case '-', '_', ' ':
			return true
		default:
			return false
		}
	})

	if len(parts) == 0 {
		parts = []string{cleaned}
	}

	for i, part := range parts {
		runes := []rune(strings.ToLower(part))
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}

	return strings.Join(parts, " ")
}
