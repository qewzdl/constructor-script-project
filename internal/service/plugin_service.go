package service

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/plugin"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/utils"
)

var (
	ErrPluginRepositoryUnavailable = errors.New("plugin repository is not configured")
	ErrPluginManagerUnavailable    = errors.New("plugin manager is not configured")
	ErrPluginNotFound              = errors.New("plugin not found")
	ErrInvalidPluginPackage        = errors.New("invalid plugin package")
)

type PluginService struct {
	mu       sync.Mutex
	repo     repository.PluginRepository
	manager  *plugin.Manager
	maxBytes int64
}

const defaultMaxPluginSize = 50 * 1024 * 1024 // 50MB

func NewPluginService(repo repository.PluginRepository, manager *plugin.Manager) *PluginService {
	if repo == nil || manager == nil {
		return nil
	}
	return &PluginService{
		repo:     repo,
		manager:  manager,
		maxBytes: defaultMaxPluginSize,
	}
}

func (s *PluginService) List() ([]models.PluginInfo, error) {
	if s == nil {
		return nil, ErrPluginManagerUnavailable
	}
	if s.repo == nil {
		return nil, ErrPluginRepositoryUnavailable
	}
	if s.manager == nil {
		return nil, ErrPluginManagerUnavailable
	}

	records, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	recordMap := make(map[string]models.Plugin, len(records))
	for _, record := range records {
		recordMap[strings.ToLower(strings.TrimSpace(record.Slug))] = record
	}

	plugins := s.manager.List()
	results := make([]models.PluginInfo, 0, len(plugins)+len(records))

	for _, entry := range plugins {
		if entry == nil {
			continue
		}

		slug := strings.ToLower(strings.TrimSpace(entry.Slug))
		record, exists := recordMap[slug]
		if exists {
			delete(recordMap, slug)
		}

		info := models.PluginInfo{
			Slug:         slug,
			Name:         entry.Metadata.Name,
			Description:  entry.Metadata.Description,
			Version:      entry.Metadata.Version,
			Author:       entry.Metadata.Author,
			Homepage:     entry.Metadata.Homepage,
			Installed:    exists,
			Active:       exists && record.Active,
			MissingFiles: false,
		}

		if exists {
			installedAt := record.InstalledAt
			info.InstalledAt = &installedAt
			if record.LastActivatedAt != nil {
				info.LastActiveAt = record.LastActivatedAt
			}
			info.AdditionalData = record.Metadata
		}

		results = append(results, info)
	}

	for slug, record := range recordMap {
		installedAt := record.InstalledAt
		info := models.PluginInfo{
			Slug:           slug,
			Name:           record.Name,
			Description:    record.Description,
			Version:        record.Version,
			Author:         record.Author,
			Homepage:       record.Homepage,
			Active:         record.Active,
			Installed:      true,
			InstalledAt:    &installedAt,
			MissingFiles:   true,
			AdditionalData: record.Metadata,
		}
		if record.LastActivatedAt != nil {
			info.LastActiveAt = record.LastActivatedAt
		}
		results = append(results, info)
	}

	sort.Slice(results, func(i, j int) bool {
		nameI := strings.TrimSpace(results[i].Name)
		nameJ := strings.TrimSpace(results[j].Name)

		if nameI != "" && nameJ != "" {
			return strings.ToLower(nameI) < strings.ToLower(nameJ)
		}

		if nameI != nameJ {
			return nameI != ""
		}

		return results[i].Slug < results[j].Slug
	})

	return results, nil
}

func (s *PluginService) Install(file io.Reader, size int64, filename string) (models.PluginInfo, error) {
	if s == nil {
		return models.PluginInfo{}, ErrPluginManagerUnavailable
	}
	if s.repo == nil {
		return models.PluginInfo{}, ErrPluginRepositoryUnavailable
	}
	if s.manager == nil {
		return models.PluginInfo{}, ErrPluginManagerUnavailable
	}

	tempFile, err := os.CreateTemp("", "plugin-*.zip")
	if err != nil {
		return models.PluginInfo{}, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	limit := s.maxBytes
	if size > 0 && (limit == 0 || size < limit) {
		limit = size
	}

	reader := file
	if limit > 0 {
		reader = io.LimitReader(file, limit+1)
	}

	written, err := io.Copy(tempFile, reader)
	if err != nil {
		return models.PluginInfo{}, fmt.Errorf("failed to store plugin archive: %w", err)
	}
	if limit > 0 && written > limit {
		return models.PluginInfo{}, fmt.Errorf("plugin package exceeds maximum size of %d bytes", limit)
	}

	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		return models.PluginInfo{}, fmt.Errorf("failed to rewind temporary file: %w", err)
	}

	archive, err := zip.NewReader(tempFile, written)
	if err != nil {
		return models.PluginInfo{}, fmt.Errorf("failed to open plugin archive: %w", err)
	}

	manifest, prefix, err := extractManifest(archive)
	if err != nil {
		return models.PluginInfo{}, err
	}

	if err := validateManifest(manifest); err != nil {
		return models.PluginInfo{}, err
	}

	slug := strings.ToLower(strings.TrimSpace(manifest.Slug))
	if slug == "" {
		slug = utils.GenerateSlug(manifest.Name)
	}
	if slug == "" {
		base := strings.TrimSuffix(filename, filepath.Ext(filename))
		slug = utils.GenerateSlug(base)
	}
	if slug == "" {
		return models.PluginInfo{}, fmt.Errorf("%w: unable to determine plugin slug", ErrInvalidPluginPackage)
	}

	destDir := filepath.Join(s.manager.BaseDir(), slug)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.RemoveAll(destDir); err != nil {
		return models.PluginInfo{}, fmt.Errorf("failed to clean plugin directory: %w", err)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return models.PluginInfo{}, fmt.Errorf("failed to prepare plugin directory: %w", err)
	}

	if err := extractArchive(archive, destDir, prefix); err != nil {
		return models.PluginInfo{}, err
	}

	if err := s.manager.Reload(); err != nil {
		return models.PluginInfo{}, fmt.Errorf("failed to reload plugins: %w", err)
	}

	pluginEntry, ok := s.manager.Resolve(slug)
	if !ok {
		return models.PluginInfo{}, fmt.Errorf("%w: %s", ErrPluginNotFound, slug)
	}

	record, err := s.repo.GetBySlug(slug)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return models.PluginInfo{}, err
		}
		record = &models.Plugin{
			Slug:        slug,
			Name:        pluginEntry.Metadata.Name,
			Description: pluginEntry.Metadata.Description,
			Version:     pluginEntry.Metadata.Version,
			Author:      pluginEntry.Metadata.Author,
			Homepage:    pluginEntry.Metadata.Homepage,
		}
	}

	metadata := make(models.JSONMap)
	if record.Metadata != nil {
		for key, value := range record.Metadata {
			metadata[key] = value
		}
	}

	metadata["manifest"] = map[string]string{
		"name":        manifest.Name,
		"slug":        manifest.Slug,
		"version":     manifest.Version,
		"description": manifest.Description,
		"author":      manifest.Author,
		"homepage":    manifest.Homepage,
	}
	metadata["resolved_slug"] = slug

	record.Name = pluginEntry.Metadata.Name
	record.Description = pluginEntry.Metadata.Description
	record.Version = pluginEntry.Metadata.Version
	record.Author = pluginEntry.Metadata.Author
	record.Homepage = pluginEntry.Metadata.Homepage
	record.Metadata = metadata

	if err := s.repo.Save(record); err != nil {
		return models.PluginInfo{}, err
	}

	installedAt := record.InstalledAt
	info := models.PluginInfo{
		Slug:           slug,
		Name:           record.Name,
		Description:    record.Description,
		Version:        record.Version,
		Author:         record.Author,
		Homepage:       record.Homepage,
		Active:         record.Active,
		Installed:      true,
		InstalledAt:    &installedAt,
		LastActiveAt:   record.LastActivatedAt,
		MissingFiles:   false,
		AdditionalData: record.Metadata,
	}

	return info, nil
}

func (s *PluginService) Activate(slug string) (models.PluginInfo, error) {
	if s == nil {
		return models.PluginInfo{}, ErrPluginManagerUnavailable
	}
	if s.repo == nil {
		return models.PluginInfo{}, ErrPluginRepositoryUnavailable
	}
	if s.manager == nil {
		return models.PluginInfo{}, ErrPluginManagerUnavailable
	}

	cleaned := strings.ToLower(strings.TrimSpace(slug))
	if cleaned == "" {
		return models.PluginInfo{}, fmt.Errorf("%w: %s", ErrPluginNotFound, slug)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.manager.Resolve(cleaned)
	if !ok {
		return models.PluginInfo{}, fmt.Errorf("%w: %s", ErrPluginNotFound, cleaned)
	}

	record, err := s.repo.GetBySlug(cleaned)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return models.PluginInfo{}, err
		}
		record = &models.Plugin{
			Slug: cleaned,
		}
	}

	now := time.Now().UTC()
	record.Name = entry.Metadata.Name
	record.Description = entry.Metadata.Description
	record.Version = entry.Metadata.Version
	record.Author = entry.Metadata.Author
	record.Homepage = entry.Metadata.Homepage
	record.Active = true
	record.LastActivatedAt = &now

	if record.Metadata == nil {
		record.Metadata = models.JSONMap{}
	}
	record.Metadata["activated_at"] = now.Format(time.RFC3339)

	if err := s.repo.Save(record); err != nil {
		return models.PluginInfo{}, err
	}

	installedAt := record.InstalledAt
	info := models.PluginInfo{
		Slug:           cleaned,
		Name:           record.Name,
		Description:    record.Description,
		Version:        record.Version,
		Author:         record.Author,
		Homepage:       record.Homepage,
		Active:         true,
		Installed:      true,
		InstalledAt:    &installedAt,
		LastActiveAt:   record.LastActivatedAt,
		MissingFiles:   false,
		AdditionalData: record.Metadata,
	}
	return info, nil
}

func (s *PluginService) Deactivate(slug string) (models.PluginInfo, error) {
	if s == nil {
		return models.PluginInfo{}, ErrPluginManagerUnavailable
	}
	if s.repo == nil {
		return models.PluginInfo{}, ErrPluginRepositoryUnavailable
	}
	if s.manager == nil {
		return models.PluginInfo{}, ErrPluginManagerUnavailable
	}

	cleaned := strings.ToLower(strings.TrimSpace(slug))
	if cleaned == "" {
		return models.PluginInfo{}, fmt.Errorf("%w: %s", ErrPluginNotFound, slug)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.manager.Resolve(cleaned)
	if !ok {
		return models.PluginInfo{}, fmt.Errorf("%w: %s", ErrPluginNotFound, cleaned)
	}

	record, err := s.repo.GetBySlug(cleaned)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			info := models.PluginInfo{
				Slug:         cleaned,
				Name:         entry.Metadata.Name,
				Description:  entry.Metadata.Description,
				Version:      entry.Metadata.Version,
				Author:       entry.Metadata.Author,
				Homepage:     entry.Metadata.Homepage,
				Active:       false,
				Installed:    false,
				MissingFiles: false,
			}
			return info, nil
		}
		return models.PluginInfo{}, err
	}

	record.Active = false
	if record.Metadata == nil {
		record.Metadata = models.JSONMap{}
	}
	record.Metadata["deactivated_at"] = time.Now().UTC().Format(time.RFC3339)

	if err := s.repo.Save(record); err != nil {
		return models.PluginInfo{}, err
	}

	installedAt := record.InstalledAt
	info := models.PluginInfo{
		Slug:           cleaned,
		Name:           record.Name,
		Description:    record.Description,
		Version:        record.Version,
		Author:         record.Author,
		Homepage:       record.Homepage,
		Active:         false,
		Installed:      true,
		InstalledAt:    &installedAt,
		LastActiveAt:   record.LastActivatedAt,
		MissingFiles:   false,
		AdditionalData: record.Metadata,
	}
	return info, nil
}

func extractManifest(reader *zip.Reader) (plugin.Metadata, string, error) {
	var manifestFile *zip.File
	var manifestPrefix string

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		cleaned := filepath.ToSlash(file.Name)
		cleaned = strings.TrimPrefix(cleaned, "./")
		if strings.EqualFold(path.Base(cleaned), "plugin.json") {
			manifestFile = file
			manifestPrefix = path.Dir(cleaned)
			break
		}
	}

	if manifestFile == nil {
		return plugin.Metadata{}, "", fmt.Errorf("%w: manifest missing", ErrInvalidPluginPackage)
	}

	rc, err := manifestFile.Open()
	if err != nil {
		return plugin.Metadata{}, "", fmt.Errorf("failed to read plugin manifest: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return plugin.Metadata{}, "", fmt.Errorf("failed to load plugin manifest: %w", err)
	}

	var manifest plugin.Metadata
	if err := json.Unmarshal(data, &manifest); err != nil {
		return plugin.Metadata{}, "", fmt.Errorf("failed to decode plugin manifest: %w", err)
	}

	manifest.Slug = strings.ToLower(strings.TrimSpace(manifest.Slug))
	manifestPrefix = strings.Trim(manifestPrefix, "/")
	return manifest, manifestPrefix, nil
}

func extractArchive(reader *zip.Reader, destDir, prefix string) error {
	for _, file := range reader.File {
		targetPath, skip := resolveTargetPath(file.Name, destDir, prefix)
		if skip {
			continue
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return fmt.Errorf("failed to create plugin directory: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("failed to create plugin directory: %w", err)
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open archive entry: %w", err)
		}

		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}

		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return fmt.Errorf("failed to write file: %w", err)
		}

		out.Close()
		rc.Close()
	}

	return nil
}

func resolveTargetPath(name, destDir, prefix string) (string, bool) {
	cleaned := filepath.ToSlash(name)
	cleaned = strings.TrimPrefix(cleaned, "./")

	if prefix != "" {
		prefixClean := strings.TrimPrefix(prefix, "./")
		prefixClean = strings.Trim(prefixClean, "/")
		if cleaned == prefixClean {
			return "", true
		}
		if !strings.HasPrefix(cleaned, prefixClean+"/") {
			return "", true
		}
		cleaned = strings.TrimPrefix(cleaned, prefixClean+"/")
	}

	cleaned = path.Clean(cleaned)
	if strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, ":") {
		return "", true
	}

	if cleaned == "" || cleaned == "." {
		return "", true
	}

	targetPath := filepath.Join(destDir, filepath.FromSlash(cleaned))
	return targetPath, false
}

func validateManifest(manifest plugin.Metadata) error {
	if strings.TrimSpace(manifest.Name) == "" {
		return fmt.Errorf("%w: plugin name is required", ErrInvalidPluginPackage)
	}
	return nil
}
