package service

import (
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
}

type UploadInfo struct {
	URL      string    `json:"url"`
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
}

var (
	ErrUploadNotFound    = errors.New("upload not found")
	ErrInvalidUploadName = errors.New("invalid image name")
)

func NewUploadService(uploadDir string) *UploadService {

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}

	return &UploadService{
		uploadDir:         uploadDir,
		maxSize:           10 * 1024 * 1024,
		allowedTypes:      []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".ico"},
		videoMaxSize:      1024 * 1024 * 1024,
		videoAllowedTypes: []string{".mp4", ".m4v", ".mov"},
	}
}

func (s *UploadService) UploadImage(file *multipart.FileHeader, preferredName string) (string, string, error) {

	if file.Size > s.maxSize {
		return "", "", errors.New("file size exceeds maximum allowed size")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !s.isAllowedType(ext, s.allowedTypes) {
		return "", "", errors.New("file type not allowed")
	}

	filename := s.generateFilename(file.Filename, preferredName, ext)
	filePath := filepath.Join(s.uploadDir, filename)

	src, err := file.Open()
	if err != nil {
		return "", "", err
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return "", "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", "", err
	}

	url := "/uploads/" + filename
	return url, filename, nil
}

func (s *UploadService) UploadMultipleImages(files []*multipart.FileHeader) ([]string, error) {
	var urls []string

	for _, file := range files {
		url, _, err := s.UploadImage(file, "")
		if err != nil {

			for _, u := range urls {
				s.DeleteImage(u)
			}
			return nil, err
		}
		urls = append(urls, url)
	}

	return urls, nil
}

func (s *UploadService) UploadVideo(file *multipart.FileHeader, preferredName string) (string, string, time.Duration, error) {
	if s == nil {
		return "", "", 0, errors.New("upload service is not configured")
	}
	if file == nil {
		return "", "", 0, errors.New("video file is required")
	}

	if file.Size > s.videoMaxSize {
		return "", "", 0, errors.New("file size exceeds maximum allowed size")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !s.isAllowedType(ext, s.videoAllowedTypes) {
		return "", "", 0, errors.New("file type not allowed")
	}

	filename := s.generateFilename(file.Filename, preferredName, ext)
	filePath := filepath.Join(s.uploadDir, filename)

	src, err := file.Open()
	if err != nil {
		return "", "", 0, err
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return "", "", 0, err
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(filePath)
		return "", "", 0, err
	}

	if err := dst.Close(); err != nil {
		os.Remove(filePath)
		return "", "", 0, err
	}

	duration, err := media.MP4Duration(filePath)
	if err != nil {
		os.Remove(filePath)
		return "", "", 0, err
	}

	url := "/uploads/" + filename
	return url, filename, duration, nil
}

func (s *UploadService) DeleteImage(url string) error {

	filename := filepath.Base(url)
	filePath := filepath.Join(s.uploadDir, filename)

	uploadDirAbs, err := filepath.Abs(s.uploadDir)
	if err != nil {
		return err
	}

	filePathAbs, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(filePathAbs, uploadDirAbs) {
		return errors.New("invalid file path")
	}

	if err := os.Remove(filePathAbs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
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

	trimmedName := strings.TrimSpace(newName)
	if trimmedName == "" {
		return UploadInfo{}, ErrInvalidUploadName
	}

	filename := filepath.Base(strings.TrimSpace(current))
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		return UploadInfo{}, ErrUploadNotFound
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !s.isAllowedType(ext, s.allowedTypes) {
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

func (s *UploadService) ListImages() ([]UploadInfo, error) {

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
		if !s.isAllowedType(ext, s.allowedTypes) {
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
		})
	}

	sort.Slice(uploads, func(i, j int) bool {
		return uploads[i].ModTime.After(uploads[j].ModTime)
	})

	return uploads, nil
}
