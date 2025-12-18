package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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

	// Try to infer metadata (mime type, file size, simple file type) from the remote URL
	if file.FileURL != "" {
		// Only attempt inference if fields are missing
		if file.MimeType == "" || file.FileSize == 0 || file.FileType == "" {
			if ct, size, err := inferRemoteFileMetadata(file.FileURL); err == nil {
				if file.MimeType == "" && ct != "" {
					file.MimeType = ct
				}
				if file.FileSize == 0 && size > 0 {
					file.FileSize = size
				}
				if file.FileType == "" {
					file.FileType = mapMimeToType(file.MimeType, file.FileURL)
				}
			}
		}
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
		// if url changed, mark for metadata inference below
		urlChanged := !strings.EqualFold(file.FileURL, url)
		file.FileURL = url
		if urlChanged {
			// clear fields so they may be inferred if not provided explicitly
			if req.MimeType == nil {
				file.MimeType = ""
			}
			if req.FileType == nil {
				file.FileType = ""
			}
			if req.FileSize == nil {
				file.FileSize = 0
			}
		}
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

	// If metadata is missing after update, try to infer from the remote URL
	if file.FileURL != "" {
		if file.MimeType == "" || file.FileSize == 0 || file.FileType == "" {
			if ct, size, err := inferRemoteFileMetadata(file.FileURL); err == nil {
				if file.MimeType == "" && ct != "" {
					file.MimeType = ct
				}
				if file.FileSize == 0 && size > 0 {
					file.FileSize = size
				}
				if file.FileType == "" {
					file.FileType = mapMimeToType(file.MimeType, file.FileURL)
				}
			}
		}
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

// inferRemoteFileMetadata attempts to fetch Content-Type and Content-Length from the
// given URL. It prefers HEAD, and falls back to a ranged GET if necessary.
func inferRemoteFileMetadata(url string) (mimeType string, size int64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp == nil || resp.StatusCode >= 400 {
		// try ranged GET (single byte) as fallback
		req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		req2.Header.Set("Range", "bytes=0-0")
		resp, err = client.Do(req2)
		if err != nil {
			return "", 0, err
		}
	}
	defer resp.Body.Close()

	ct := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if ct != "" {
		// strip any charset
		if idx := strings.Index(ct, ";"); idx >= 0 {
			ct = strings.TrimSpace(ct[:idx])
		}
		mimeType = strings.ToLower(ct)
	}

	cl := strings.TrimSpace(resp.Header.Get("Content-Length"))
	if cl != "" {
		if v, err := strconv.ParseInt(cl, 10, 64); err == nil {
			size = v
		}
	}

	// Some ranged responses use Content-Range instead of Content-Length
	if size == 0 {
		cr := strings.TrimSpace(resp.Header.Get("Content-Range"))
		// format: bytes 0-0/12345
		if cr != "" {
			if slash := strings.LastIndex(cr, "/"); slash >= 0 && slash+1 < len(cr) {
				if v, err := strconv.ParseInt(strings.TrimSpace(cr[slash+1:]), 10, 64); err == nil {
					size = v
				}
			}
		}
	}

	return mimeType, size, nil
}

// mapMimeToType produces a simple human-friendly file type label from a mime type
// or URL extension.
func mapMimeToType(mimeType, url string) string {
	mt := strings.ToLower(strings.TrimSpace(mimeType))
	if mt != "" {
		switch {
		case strings.HasPrefix(mt, "image/"):
			return "Image"
		case strings.HasPrefix(mt, "video/"):
			return "Video"
		case strings.HasPrefix(mt, "audio/"):
			return "Audio"
		case mt == "application/pdf":
			return "Document"
		case strings.HasPrefix(mt, "text/"):
			return "Document"
		case strings.HasPrefix(mt, "application/zip") || strings.Contains(mt, "compressed") || strings.Contains(mt, "tar"):
			return "Archive"
		}
	}
	// fallback to extension-based hints
	lower := strings.ToLower(url)
	if strings.HasSuffix(lower, ".pdf") || strings.HasSuffix(lower, ".doc") || strings.HasSuffix(lower, ".docx") || strings.HasSuffix(lower, ".txt") {
		return "Document"
	}
	if strings.HasSuffix(lower, ".zip") || strings.HasSuffix(lower, ".tar") || strings.HasSuffix(lower, ".gz") || strings.HasSuffix(lower, ".7z") {
		return "Archive"
	}
	if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".gif") || strings.HasSuffix(lower, ".svg") {
		return "Image"
	}
	if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".mov") || strings.HasSuffix(lower, ".webm") {
		return "Video"
	}
	return "Other"
}
