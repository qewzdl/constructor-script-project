package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/utils"
)

type FileService struct {
	fileRepo         repository.ArchiveFileRepository
	directoryRepo    repository.ArchiveDirectoryRepository
	directoryService *DirectoryService
}

func NewFileService(fileRepo repository.ArchiveFileRepository, directoryRepo repository.ArchiveDirectoryRepository, directoryService *DirectoryService) *FileService {
	return &FileService{
		fileRepo:         fileRepo,
		directoryRepo:    directoryRepo,
		directoryService: directoryService,
	}
}

func (s *FileService) Create(req models.CreateArchiveFileRequest) (*models.ArchiveFile, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("file name is required")
	}

	fileURL := strings.TrimSpace(req.FileURL)
	if fileURL == "" {
		return nil, fmt.Errorf("file url is required")
	}

	directory, err := s.directoryRepo.GetByID(req.DirectoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDirectoryNotFound
		}
		return nil, err
	}

	slug := sanitizeDirectorySlug(req.Slug)
	if slug == "" {
		slug = utils.GenerateSlug(name)
	}
	if slug == "" {
		slug = fmt.Sprintf("file-%d", time.Now().UnixNano())
	}

	uniqueSlug, err := s.ensureFileSlug(directory.ID, slug, nil)
	if err != nil {
		return nil, err
	}

	file := &models.ArchiveFile{
		DirectoryID: directory.ID,
		Name:        name,
		Slug:        uniqueSlug,
		Path:        buildFilePath(directory.Path, uniqueSlug),
		Description: strings.TrimSpace(req.Description),
		FileURL:     fileURL,
		PreviewURL:  strings.TrimSpace(req.PreviewURL),
		MimeType:    strings.TrimSpace(req.MimeType),
		FileType:    strings.TrimSpace(req.FileType),
		Published:   req.Published,
		Order:       req.Order,
	}
	if req.FileSize != nil {
		file.FileSize = *req.FileSize
	}

	if err := s.fileRepo.Create(file); err != nil {
		return nil, err
	}

	s.invalidateTreeCache()
	return file, nil
}

func (s *FileService) Update(id uint, req models.UpdateArchiveFileRequest) (*models.ArchiveFile, error) {
	file, err := s.fileRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	originalDirectoryID := file.DirectoryID
	originalSlug := file.Slug

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("file name cannot be empty")
		}
		file.Name = name
	}

	if req.Description != nil {
		file.Description = strings.TrimSpace(*req.Description)
	}

	if req.FileURL != nil {
		url := strings.TrimSpace(*req.FileURL)
		if url == "" {
			return nil, fmt.Errorf("file url cannot be empty")
		}
		file.FileURL = url
	}

	if req.PreviewURL != nil {
		file.PreviewURL = strings.TrimSpace(*req.PreviewURL)
	}

	if req.MimeType != nil {
		file.MimeType = strings.TrimSpace(*req.MimeType)
	}

	if req.FileType != nil {
		file.FileType = strings.TrimSpace(*req.FileType)
	}

	if req.FileSize != nil {
		file.FileSize = *req.FileSize
	}

	if req.Published != nil {
		file.Published = *req.Published
	}

	if req.Order != nil {
		file.Order = *req.Order
	}

	if req.DirectoryID.Set {
		if req.DirectoryID.Value == nil {
			return nil, ErrDirectoryNotFound
		}
		newDirectoryID := *req.DirectoryID.Value
		if newDirectoryID != file.DirectoryID {
			directory, err := s.directoryRepo.GetByID(newDirectoryID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, ErrDirectoryNotFound
				}
				return nil, err
			}
			file.DirectoryID = directory.ID
			file.Path = buildFilePath(directory.Path, file.Slug)
		}
	}

	if req.Slug != nil {
		provided := sanitizeDirectorySlug(*req.Slug)
		if provided == "" && file.Name != "" {
			provided = utils.GenerateSlug(file.Name)
		}
		if provided == "" {
			provided = fmt.Sprintf("file-%d", file.ID)
		}
		if !strings.EqualFold(provided, file.Slug) {
			file.Slug = provided
		}
	}

	if file.Slug == "" {
		generated := utils.GenerateSlug(file.Name)
		if generated == "" {
			generated = fmt.Sprintf("file-%d", file.ID)
		}
		file.Slug = generated
	}

	if file.DirectoryID != originalDirectoryID || !strings.EqualFold(file.Slug, originalSlug) {
		uniqueSlug, err := s.ensureFileSlug(file.DirectoryID, file.Slug, &file.ID)
		if err != nil {
			return nil, err
		}
		file.Slug = uniqueSlug

		directory, err := s.directoryRepo.GetByID(file.DirectoryID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrDirectoryNotFound
			}
			return nil, err
		}
		file.Path = buildFilePath(directory.Path, file.Slug)
	} else {
		directory, err := s.directoryRepo.GetByID(file.DirectoryID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrDirectoryNotFound
			}
			return nil, err
		}
		file.Path = buildFilePath(directory.Path, file.Slug)
	}

	if err := s.fileRepo.Update(file); err != nil {
		return nil, err
	}

	s.invalidateTreeCache()
	return file, nil
}

func (s *FileService) Delete(id uint) error {
	if _, err := s.fileRepo.GetByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFileNotFound
		}
		return err
	}
	if err := s.fileRepo.Delete(id); err != nil {
		return err
	}
	s.invalidateTreeCache()
	return nil
}

func (s *FileService) GetByID(id uint, includeUnpublished bool) (*models.ArchiveFile, error) {
	file, err := s.fileRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	if !includeUnpublished && !file.Published {
		return nil, ErrFileNotFound
	}
	return file, nil
}

func (s *FileService) GetByPath(path string, includeUnpublished bool) (*models.ArchiveFile, error) {
	file, err := s.fileRepo.GetByPath(path)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	if !includeUnpublished && !file.Published {
		return nil, ErrFileNotFound
	}
	return file, nil
}

func (s *FileService) ListByDirectory(directoryID uint, includeUnpublished bool) ([]models.ArchiveFile, error) {
	files, err := s.fileRepo.ListByDirectory(directoryID, includeUnpublished)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (s *FileService) ListByDirectoryPath(path string, includeUnpublished bool) ([]models.ArchiveFile, *models.ArchiveDirectory, error) {
	directory, err := s.directoryRepo.GetByPath(path)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrDirectoryNotFound
		}
		return nil, nil, err
	}
	if !includeUnpublished && !directory.Published {
		return nil, nil, ErrDirectoryNotFound
	}
	files, err := s.fileRepo.ListByDirectory(directory.ID, includeUnpublished)
	if err != nil {
		return nil, nil, err
	}
	return files, directory, nil
}

func (s *FileService) ensureFileSlug(directoryID uint, base string, excludeID *uint) (string, error) {
	candidate := strings.TrimSpace(strings.ToLower(base))
	if candidate == "" {
		candidate = "file"
	}
	baseSlug := candidate
	suffix := 1
	for {
		exists, err := s.fileRepo.ExistsBySlug(directoryID, candidate, excludeID)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", baseSlug, suffix)
		suffix++
	}
}

func (s *FileService) invalidateTreeCache() {
	if s.directoryService != nil {
		s.directoryService.invalidateTreeCache()
	}
}
