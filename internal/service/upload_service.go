package service

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"constructor-script-backend/pkg/validator"
)

type UploadService struct {
	uploadDir    string
	maxSize      int64
	allowedTypes []string
}

func NewUploadService(uploadDir string) *UploadService {

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}

	return &UploadService{
		uploadDir:    uploadDir,
		maxSize:      10 * 1024 * 1024,
		allowedTypes: []string{".jpg", ".jpeg", ".png", ".gif", ".webp"},
	}
}

func (s *UploadService) UploadImage(file *multipart.FileHeader) (string, error) {

	if file.Size > s.maxSize {
		return "", errors.New("file size exceeds maximum allowed size")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !s.isAllowedType(ext) {
		return "", errors.New("file type not allowed")
	}

	filename := s.generateFilename(ext)
	filePath := filepath.Join(s.uploadDir, filename)

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	url := "/uploads/" + filename
	return url, nil
}

func (s *UploadService) UploadMultipleImages(files []*multipart.FileHeader) ([]string, error) {
	var urls []string

	for _, file := range files {
		url, err := s.UploadImage(file)
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

func (s *UploadService) DeleteImage(url string) error {

	filename := filepath.Base(url)
	filePath := filepath.Join(s.uploadDir, filename)

	if !strings.HasPrefix(filePath, s.uploadDir) {
		return errors.New("invalid file path")
	}

	return os.Remove(filePath)
}

func (s *UploadService) isAllowedType(ext string) bool {
	for _, allowedExt := range s.allowedTypes {
		if ext == allowedExt {
			return true
		}
	}
	return false
}

func (s *UploadService) generateFilename(ext string) string {
	return fmt.Sprintf("%s%s", uuid.New().String(), ext)
}

func (s *UploadService) GetFileInfo(url string) (os.FileInfo, error) {
	filename := filepath.Base(url)
	filePath := filepath.Join(s.uploadDir, filename)
	return os.Stat(filePath)
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
