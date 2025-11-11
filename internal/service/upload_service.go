package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"constructor-script-backend/pkg/media"
	"constructor-script-backend/pkg/utils"
	"constructor-script-backend/pkg/validator"
)

type UploadService struct {
	uploadDir         string
	maxSize           int64
	allowedTypes      []string
	videoMaxSize      int64
	videoAllowedTypes []string
	fileMaxSize       int64
	fileAllowedTypes  []string
	subtitleManager   *SubtitleManager
	subtitleConfig    SubtitleGenerationConfig
}

type UploadInfo struct {
	URL      string    `json:"url"`
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
	Type     string    `json:"type"`
}

// SubtitleGenerationConfig captures the defaults applied when the upload service
// requests subtitles from the configured manager.
type SubtitleGenerationConfig struct {
	Provider      string
	PreferredName string
	Language      string
	Prompt        string
	Temperature   *float32
}

// VideoUploadResult captures metadata for an uploaded video and its derived assets.
type VideoUploadResult struct {
	Video    UploadInfo    `json:"video"`
	Duration time.Duration `json:"duration"`
	Subtitle *UploadInfo   `json:"subtitle,omitempty"`
}

type UploadCategory string

const (
	UploadCategoryImage UploadCategory = "image"
	UploadCategoryVideo UploadCategory = "video"
	UploadCategoryFile  UploadCategory = "file"
)

var (
	ErrUploadNotFound       = errors.New("upload not found")
	ErrInvalidUploadName    = errors.New("invalid upload name")
	ErrUnsupportedUpload    = errors.New("file type not allowed")
	ErrUploadTooLarge       = errors.New("file size exceeds maximum allowed size")
	ErrUploadMissing        = errors.New("file is required")
	ErrSubtitleContentEmpty = errors.New("subtitle content is required")
	errUploadServiceMissing = errors.New("upload service is not configured")
)

func NewUploadService(uploadDir string) *UploadService {

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}

	return &UploadService{
		uploadDir:         uploadDir,
		maxSize:           10 * 1024 * 1024,
		allowedTypes:      []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".ico", ".svg"},
		videoMaxSize:      1024 * 1024 * 1024,
		videoAllowedTypes: []string{".mp4", ".m4v", ".mov"},
		fileMaxSize:       50 * 1024 * 1024,
		fileAllowedTypes: []string{
			".pdf", ".txt", ".csv", ".json", ".xml", ".md", ".vtt",
			".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
			".zip", ".tar", ".gz", ".tgz", ".rar", ".7z",
		},
	}
}

// SetSubtitleGenerator attaches a subtitle generator that will run for each uploaded video.
func (s *UploadService) SetSubtitleGenerator(generator SubtitleGenerator) {
	if s == nil {
		return
	}

	if generator == nil {
		s.subtitleManager = nil
		return
	}

	manager := NewSubtitleManager("default")
	if err := manager.Register("default", generator); err != nil {
		return
	}
	s.subtitleManager = manager
	s.subtitleConfig.Provider = "default"
}

// UseSubtitleManager configures a shared subtitle manager instance for uploads.
func (s *UploadService) UseSubtitleManager(manager *SubtitleManager) {
	if s == nil {
		return
	}
	s.subtitleManager = manager
}

// ConfigureSubtitleGeneration adjusts the default subtitle request that will be
// used when invoking the subtitle manager.
func (s *UploadService) ConfigureSubtitleGeneration(config SubtitleGenerationConfig) {
	if s == nil {
		return
	}
	s.subtitleConfig = config
}

func (s *UploadService) Upload(file *multipart.FileHeader, preferredName string) (UploadInfo, error) {
	if s == nil {
		return UploadInfo{}, errUploadServiceMissing
	}
	if file == nil {
		return UploadInfo{}, ErrUploadMissing
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))

	switch {
	case s.isAllowedType(ext, s.allowedTypes):
		return s.uploadImage(file, preferredName)
	case s.isAllowedType(ext, s.videoAllowedTypes):
		result, err := s.uploadVideo(context.Background(), file, preferredName)
		return result.Video, err
	case s.isAllowedType(ext, s.fileAllowedTypes):
		return s.uploadDocument(file, preferredName)
	default:
		return UploadInfo{}, ErrUnsupportedUpload
	}
}

func (s *UploadService) UploadImage(file *multipart.FileHeader, preferredName string) (string, string, error) {
	info, err := s.uploadImage(file, preferredName)
	if err != nil {
		return "", "", err
	}
	return info.URL, info.Filename, nil
}

func (s *UploadService) UploadMultipleImages(files []*multipart.FileHeader) ([]string, error) {
	var urls []string

	for _, file := range files {
		info, err := s.uploadImage(file, "")
		if err != nil {
			for _, u := range urls {
				s.DeleteImage(u)
			}
			return nil, err
		}
		urls = append(urls, info.URL)
	}

	return urls, nil
}

func (s *UploadService) UploadVideo(ctx context.Context, file *multipart.FileHeader, preferredName string) (VideoUploadResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	return s.uploadVideo(ctx, file, preferredName)
}

// SaveSubtitle persists subtitle content for a video and returns the stored asset metadata.
func (s *UploadService) SaveSubtitle(videoFilename string, content []byte, preferredName string) (*UploadInfo, error) {
	if s == nil {
		return nil, errUploadServiceMissing
	}
	if len(bytes.TrimSpace(content)) == 0 {
		return nil, ErrSubtitleContentEmpty
	}

	payload := make([]byte, len(content))
	copy(payload, content)

	result := &SubtitleResult{
		Format: SubtitleFormatVTT,
		Data:   payload,
	}
	if name := strings.TrimSpace(preferredName); name != "" {
		result.Name = name
	}

	info, _, err := s.persistSubtitle(videoFilename, result)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func (s *UploadService) DeleteImage(url string) error {

	if s == nil {
		return errUploadServiceMissing
	}

	if err := s.DeleteUpload(url); err != nil {
		if errors.Is(err, ErrUploadNotFound) {
			return nil
		}
		return err
	}

	return nil
}

func (s *UploadService) DeleteUpload(current string) error {
	if s == nil {
		return errUploadServiceMissing
	}

	trimmed := strings.TrimSpace(current)
	if trimmed == "" {
		return ErrUploadNotFound
	}

	filename := filepath.Base(trimmed)
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		return ErrUploadNotFound
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := s.detectCategory(ext); !ok {
		return ErrUploadNotFound
	}

	uploadDirAbs, err := filepath.Abs(s.uploadDir)
	if err != nil {
		return err
	}

	targetPath := filepath.Join(s.uploadDir, filename)
	targetAbs, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(targetAbs, uploadDirAbs) {
		return ErrUploadNotFound
	}

	if err := os.Remove(targetAbs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrUploadNotFound
		}
		return err
	}

	return nil
}

func (s *UploadService) isAllowedType(ext string, allowed []string) bool {
	for _, allowedExt := range allowed {
		if ext == allowedExt {
			return true
		}
	}
	return false
}

func (s *UploadService) generateFilename(originalName, preferredName, ext string) string {

	baseName := strings.TrimSpace(preferredName)
	if baseName == "" {
		baseName = strings.TrimSuffix(filepath.Base(originalName), filepath.Ext(originalName))
	}

	cleaned := utils.GenerateSlug(baseName)
	if cleaned == "" {
		cleaned = uuid.New().String()
	}

	candidate := fmt.Sprintf("%s%s", cleaned, ext)
	if !s.fileExists(candidate) {
		return candidate
	}

	for i := 1; i < 1000; i++ {
		candidate = fmt.Sprintf("%s-%d%s", cleaned, i, ext)
		if !s.fileExists(candidate) {
			return candidate
		}
	}

	return fmt.Sprintf("%s%s", uuid.New().String(), ext)
}

func (s *UploadService) generateFilenameForRename(preferredName, ext, currentFilename string) string {

	cleaned := utils.GenerateSlug(preferredName)
	if cleaned == "" {
		cleaned = uuid.New().String()
	}

	candidate := fmt.Sprintf("%s%s", cleaned, ext)
	if candidate == currentFilename {
		return candidate
	}

	if !s.fileExists(candidate) {
		return candidate
	}

	for i := 1; i < 1000; i++ {
		candidate = fmt.Sprintf("%s-%d%s", cleaned, i, ext)
		if candidate == currentFilename {
			return candidate
		}
		if !s.fileExists(candidate) {
			return candidate
		}
	}

	fallback := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	if fallback == currentFilename {
		fallback = fmt.Sprintf("%s-%s%s", uuid.New().String(), uuid.New().String(), ext)
	}

	return fallback
}

func (s *UploadService) fileExists(name string) bool {
	path := filepath.Join(s.uploadDir, name)
	_, err := os.Stat(path)
	return err == nil
}

func (s *UploadService) GetFileInfo(url string) (os.FileInfo, error) {
	filename := filepath.Base(url)
	filePath := filepath.Join(s.uploadDir, filename)
	return os.Stat(filePath)
}

func (s *UploadService) RenameImage(current string, newName string) (UploadInfo, error) {
	info, err := s.RenameUpload(current, newName)
	if err != nil {
		return UploadInfo{}, err
	}
	if info.Type != string(UploadCategoryImage) {
		return UploadInfo{}, ErrUploadNotFound
	}
	return info, nil
}

func (s *UploadService) RenameUpload(current string, newName string) (UploadInfo, error) {
	trimmedName := strings.TrimSpace(newName)
	if trimmedName == "" {
		return UploadInfo{}, ErrInvalidUploadName
	}

	filename := filepath.Base(strings.TrimSpace(current))
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		return UploadInfo{}, ErrUploadNotFound
	}

	ext := strings.ToLower(filepath.Ext(filename))
	category, ok := s.detectCategory(ext)
	if !ok {
		return UploadInfo{}, ErrUploadNotFound
	}

	uploadDirAbs, err := filepath.Abs(s.uploadDir)
	if err != nil {
		return UploadInfo{}, err
	}

	currentPath := filepath.Join(s.uploadDir, filename)
	currentAbs, err := filepath.Abs(currentPath)
	if err != nil {
		return UploadInfo{}, err
	}

	if !strings.HasPrefix(currentAbs, uploadDirAbs) {
		return UploadInfo{}, ErrUploadNotFound
	}

	if _, err := os.Stat(currentAbs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return UploadInfo{}, ErrUploadNotFound
		}
		return UploadInfo{}, err
	}

	newFilename := s.generateFilenameForRename(trimmedName, ext, filename)
	if newFilename == "" {
		return UploadInfo{}, ErrInvalidUploadName
	}

	if newFilename == filename {
		info, err := os.Stat(currentAbs)
		if err != nil {
			return UploadInfo{}, err
		}

		return UploadInfo{
			URL:      "/uploads/" + filename,
			Filename: filename,
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			Type:     string(category),
		}, nil
	}

	newPath := filepath.Join(s.uploadDir, newFilename)
	newAbs, err := filepath.Abs(newPath)
	if err != nil {
		return UploadInfo{}, err
	}

	if !strings.HasPrefix(newAbs, uploadDirAbs) {
		return UploadInfo{}, ErrInvalidUploadName
	}

	if err := os.Rename(currentAbs, newAbs); err != nil {
		return UploadInfo{}, err
	}

	info, err := os.Stat(newAbs)
	if err != nil {
		return UploadInfo{}, err
	}

	return UploadInfo{
		URL:      "/uploads/" + newFilename,
		Filename: newFilename,
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		Type:     string(category),
	}, nil
}

func (s *UploadService) ValidateImage(file *multipart.FileHeader) error {

	if !validator.ValidateFileSize(file.Size, s.maxSize) {
		return errors.New("file size is invalid")
	}

	if !validator.ValidateImageExtension(file.Filename) {
		return errors.New("invalid image format")
	}

	return nil
}

func (s *UploadService) IsManagedURL(url string) bool {
	if url == "" {
		return false
	}

	trimmed := strings.TrimSpace(url)
	return strings.HasPrefix(trimmed, "/uploads/")
}

func (s *UploadService) ListUploads() ([]UploadInfo, error) {
	entries, err := os.ReadDir(s.uploadDir)
	if err != nil {
		return nil, err
	}

	uploads := make([]UploadInfo, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		category, ok := s.detectCategory(ext)
		if !ok {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		uploads = append(uploads, UploadInfo{
			URL:      "/uploads/" + name,
			Filename: name,
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			Type:     string(category),
		})
	}

	sort.Slice(uploads, func(i, j int) bool {
		return uploads[i].ModTime.After(uploads[j].ModTime)
	})

	return uploads, nil
}

func (s *UploadService) ListImages() ([]UploadInfo, error) {
	uploads, err := s.ListUploads()
	if err != nil {
		return nil, err
	}

	images := make([]UploadInfo, 0, len(uploads))
	for _, upload := range uploads {
		if upload.Type == string(UploadCategoryImage) {
			images = append(images, upload)
		}
	}

	return images, nil
}

func (s *UploadService) uploadImage(file *multipart.FileHeader, preferredName string) (UploadInfo, error) {
	if file == nil {
		return UploadInfo{}, ErrUploadMissing
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !s.isAllowedType(ext, s.allowedTypes) {
		return UploadInfo{}, ErrUnsupportedUpload
	}

	info, _, err := s.persistUpload(file, preferredName, ext, s.maxSize, UploadCategoryImage)
	return info, err
}

func (s *UploadService) uploadVideo(ctx context.Context, file *multipart.FileHeader, preferredName string) (VideoUploadResult, error) {
	if s == nil {
		return VideoUploadResult{}, errUploadServiceMissing
	}
	if file == nil {
		return VideoUploadResult{}, ErrUploadMissing
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !s.isAllowedType(ext, s.videoAllowedTypes) {
		return VideoUploadResult{}, ErrUnsupportedUpload
	}

	upload, filePath, err := s.persistUpload(file, preferredName, ext, s.videoMaxSize, UploadCategoryVideo)
	if err != nil {
		return VideoUploadResult{}, err
	}

	cleanup := func() {
		os.Remove(filePath)
	}

	duration, err := media.MP4Duration(filePath)
	if err != nil {
		cleanup()
		return VideoUploadResult{}, err
	}

	result := VideoUploadResult{
		Video:    upload,
		Duration: duration,
	}

	if s.subtitleManager != nil {
		request := SubtitleGenerationRequest{
			SourcePath:    filePath,
			Provider:      s.subtitleConfig.Provider,
			PreferredName: s.subtitleConfig.PreferredName,
			Language:      s.subtitleConfig.Language,
			Prompt:        s.subtitleConfig.Prompt,
		}
		if s.subtitleConfig.Temperature != nil {
			value := *s.subtitleConfig.Temperature
			request.Temperature = &value
		}

		subtitleResult, err := s.subtitleManager.Generate(ctx, request)
		if err != nil {
			cleanup()
			return VideoUploadResult{}, fmt.Errorf("failed to generate subtitles: %w", err)
		}
		if subtitleResult != nil && len(subtitleResult.Data) > 0 {
			info, subtitlePath, err := s.persistSubtitle(upload.Filename, subtitleResult)
			if err != nil {
				cleanup()
				if subtitlePath != "" {
					os.Remove(subtitlePath)
				}
				return VideoUploadResult{}, fmt.Errorf("failed to persist generated subtitles: %w", err)
			}
			result.Subtitle = info
		}
	}

	return result, nil
}

func (s *UploadService) uploadDocument(file *multipart.FileHeader, preferredName string) (UploadInfo, error) {
	if file == nil {
		return UploadInfo{}, ErrUploadMissing
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !s.isAllowedType(ext, s.fileAllowedTypes) {
		return UploadInfo{}, ErrUnsupportedUpload
	}

	info, _, err := s.persistUpload(file, preferredName, ext, s.fileMaxSize, UploadCategoryFile)
	return info, err
}

func (s *UploadService) persistUpload(file *multipart.FileHeader, preferredName string, ext string, maxSize int64, category UploadCategory) (UploadInfo, string, error) {
	if s == nil {
		return UploadInfo{}, "", errUploadServiceMissing
	}

	if file.Size > maxSize {
		return UploadInfo{}, "", ErrUploadTooLarge
	}

	filename := s.generateFilename(file.Filename, preferredName, ext)
	filePath := filepath.Join(s.uploadDir, filename)

	src, err := file.Open()
	if err != nil {
		return UploadInfo{}, "", err
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return UploadInfo{}, "", err
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(filePath)
		return UploadInfo{}, "", err
	}

	if err := dst.Close(); err != nil {
		os.Remove(filePath)
		return UploadInfo{}, "", err
	}

	info, err := os.Stat(filePath)
	if err != nil {
		os.Remove(filePath)
		return UploadInfo{}, "", err
	}

	upload := UploadInfo{
		URL:      "/uploads/" + filename,
		Filename: filename,
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		Type:     string(category),
	}

	return upload, filePath, nil
}

func (s *UploadService) persistSubtitle(videoFilename string, subtitle *SubtitleResult) (*UploadInfo, string, error) {
	if s == nil {
		return nil, "", errUploadServiceMissing
	}
	if subtitle == nil || len(subtitle.Data) == 0 {
		return nil, "", nil
	}

	format := strings.TrimSpace(string(subtitle.Format))
	ext := ".vtt"
	if format != "" {
		normalized := strings.ToLower(format)
		if normalized != "vtt" && normalized != string(SubtitleFormatVTT) {
			return nil, "", fmt.Errorf("unsupported subtitle format: %s", subtitle.Format)
		}
	}

	baseName := strings.TrimSuffix(videoFilename, filepath.Ext(videoFilename))
	preferredName := baseName + " subtitles"
	if name := strings.TrimSpace(subtitle.Name); name != "" {
		preferredName = name
	}

	filename := s.generateFilename(videoFilename, preferredName, ext)
	subtitlePath := filepath.Join(s.uploadDir, filename)

	if err := os.WriteFile(subtitlePath, subtitle.Data, 0644); err != nil {
		return nil, "", err
	}

	info, err := os.Stat(subtitlePath)
	if err != nil {
		os.Remove(subtitlePath)
		return nil, "", err
	}

	upload := &UploadInfo{
		URL:      "/uploads/" + filename,
		Filename: filename,
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		Type:     string(UploadCategoryFile),
	}

	return upload, subtitlePath, nil
}

func (s *UploadService) detectCategory(ext string) (UploadCategory, bool) {
	switch {
	case s.isAllowedType(ext, s.allowedTypes):
		return UploadCategoryImage, true
	case s.isAllowedType(ext, s.videoAllowedTypes):
		return UploadCategoryVideo, true
	case s.isAllowedType(ext, s.fileAllowedTypes):
		return UploadCategoryFile, true
	default:
		return "", false
	}
}
