package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
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

	// Попытка определить метаданные из удаленного URL
	needsInference := file.MimeType == "" || file.FileSize == 0 || file.FileType == ""
	if file.FileURL != "" && needsInference {
		log.Printf("[FileService] Attempting to infer metadata for URL: %s", file.FileURL)
		if inferredMime, inferredSize, err := inferRemoteFileMetadata(file.FileURL); err == nil {
			// Обновляем MimeType если не был указан
			if file.MimeType == "" && inferredMime != "" {
				file.MimeType = inferredMime
				log.Printf("[FileService] Inferred MimeType: %s", inferredMime)
			}
			// Обновляем FileSize если не был указан
			if file.FileSize == 0 && inferredSize > 0 {
				file.FileSize = inferredSize
				log.Printf("[FileService] Inferred FileSize: %d", inferredSize)
			}
		} else {
			log.Printf("[FileService] Failed to infer metadata: %v", err)
		}
	}

	// Определяем FileType на основе доступных данных
	if file.FileType == "" {
		file.FileType = mapMimeToType(file.MimeType, file.FileURL)
		log.Printf("[FileService] Determined FileType: %s (from mime=%s, url=%s)", file.FileType, file.MimeType, file.FileURL)
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
	urlChanged := false
	shouldInferMetadata := false

	// Обновляем поля из запроса
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
		urlChanged = !strings.EqualFold(file.FileURL, url)
		file.FileURL = url

		// Если URL изменился и соответствующие поля не переданы, очищаем их
		if urlChanged {
			log.Printf("[FileService] URL changed for file %d, resetting metadata fields", id)
			if req.MimeType == nil {
				file.MimeType = ""
				shouldInferMetadata = true
			}
			if req.FileType == nil {
				file.FileType = ""
				shouldInferMetadata = true
			}
			if req.FileSize == nil {
				file.FileSize = 0
				shouldInferMetadata = true
			}
		}
	}

	if req.PreviewURL != nil {
		file.PreviewURL = strings.TrimSpace(*req.PreviewURL)
	}

	// Применяем явно переданные значения
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

	// Проверяем, нужно ли определить метаданные
	needsInference := file.MimeType == "" || file.FileSize == 0 || file.FileType == ""
	if file.FileURL != "" && (shouldInferMetadata || needsInference) {
		log.Printf("[FileService] Attempting to infer metadata for file %d (URL: %s)", id, file.FileURL)
		if inferredMime, inferredSize, err := inferRemoteFileMetadata(file.FileURL); err == nil {
			// Обновляем только пустые поля
			if file.MimeType == "" && inferredMime != "" {
				file.MimeType = inferredMime
				log.Printf("[FileService] Inferred MimeType: %s", inferredMime)
			}
			if file.FileSize == 0 && inferredSize > 0 {
				file.FileSize = inferredSize
				log.Printf("[FileService] Inferred FileSize: %d", inferredSize)
			}
		} else {
			log.Printf("[FileService] Failed to infer metadata: %v", err)
		}
	}

	// Определяем FileType если он пустой
	if file.FileType == "" {
		file.FileType = mapMimeToType(file.MimeType, file.FileURL)
		log.Printf("[FileService] Determined FileType: %s (from mime=%s, url=%s)", file.FileType, file.MimeType, file.FileURL)
	}

	// Обработка изменения директории
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
		}
	}

	// Обработка slug
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

	// Проверяем уникальность slug и обновляем path
	if file.DirectoryID != originalDirectoryID || !strings.EqualFold(file.Slug, originalSlug) {
		uniqueSlug, err := s.ensureFileSlug(file.DirectoryID, file.Slug, &file.ID)
		if err != nil {
			return nil, err
		}
		file.Slug = uniqueSlug
	}

	// Обновляем path
	directory, err := s.directoryRepo.GetByID(file.DirectoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDirectoryNotFound
		}
		return nil, err
	}
	file.Path = buildFilePath(directory.Path, file.Slug)

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

func inferRemoteFileMetadata(url string) (mimeType string, size int64, err error) {
	// Handle local filesystem paths (absolute or file://)
	if strings.HasPrefix(url, "/") || strings.HasPrefix(url, "file://") {
		var localPath string
		if strings.HasPrefix(url, "file://") {
			localPath = strings.TrimPrefix(url, "file://")
		} else {
			localPath = url
		}
		localPath = filepath.FromSlash(localPath)

		info, statErr := os.Stat(localPath)
		if statErr != nil {
			log.Printf("[inferRemoteFileMetadata] Local file stat error for %s: %v", localPath, statErr)
			return "", 0, fmt.Errorf("local file not accessible: %w", statErr)
		}
		if info.IsDir() {
			return "", 0, fmt.Errorf("local path is a directory")
		}

		size = info.Size()

		// Determine mime type by extension first
		if ext := strings.ToLower(filepath.Ext(localPath)); ext != "" {
			if mt := mime.TypeByExtension(ext); mt != "" {
				if idx := strings.Index(mt, ";"); idx >= 0 {
					mt = strings.TrimSpace(mt[:idx])
				}
				mimeType = strings.ToLower(mt)
			}
		}

		// Fallback to sniffing content if mime not found
		if mimeType == "" {
			f, openErr := os.Open(localPath)
			if openErr == nil {
				buf := make([]byte, 512)
				n, _ := f.Read(buf)
				sniff := http.DetectContentType(buf[:n])
				if sniff != "" {
					if idx := strings.Index(sniff, ";"); idx >= 0 {
						sniff = strings.TrimSpace(sniff[:idx])
					}
					mimeType = strings.ToLower(sniff)
				}
				f.Close()
			}
		}

		log.Printf("[inferRemoteFileMetadata] Local file %s: mime=%s, size=%d", localPath, mimeType, size)
		return mimeType, size, nil
	}

	// Handle remote URLs
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
		Timeout: 15 * time.Second,
	}

	// Try HEAD request first
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create HEAD request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)

	// If HEAD fails, try GET with Range
	if err != nil || resp == nil || resp.StatusCode >= 400 {
		if resp != nil {
			resp.Body.Close()
			log.Printf("[inferRemoteFileMetadata] HEAD failed for %s: status=%d", url, resp.StatusCode)
		} else {
			log.Printf("[inferRemoteFileMetadata] HEAD failed for %s: %v", url, err)
		}

		req2, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", 0, fmt.Errorf("failed to create GET request: %w", err)
		}
		req2.Header.Set("Range", "bytes=0-0")
		req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

		resp, err = client.Do(req2)
		if err != nil {
			log.Printf("[inferRemoteFileMetadata] GET also failed for %s: %v", url, err)
			return "", 0, fmt.Errorf("both HEAD and GET failed: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", 0, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Extract Content-Type
	ct := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if ct != "" {
		if idx := strings.Index(ct, ";"); idx >= 0 {
			ct = strings.TrimSpace(ct[:idx])
		}
		mimeType = strings.ToLower(ct)
	}

	// Extract Content-Length
	cl := strings.TrimSpace(resp.Header.Get("Content-Length"))
	if cl != "" {
		if v, parseErr := strconv.ParseInt(cl, 10, 64); parseErr == nil && v > 0 {
			size = v
		}
	}

	// Try Content-Range if Content-Length is not available
	if size == 0 {
		cr := strings.TrimSpace(resp.Header.Get("Content-Range"))
		if cr != "" {
			// Content-Range format: bytes 0-0/12345
			if slash := strings.LastIndex(cr, "/"); slash >= 0 && slash+1 < len(cr) {
				totalStr := strings.TrimSpace(cr[slash+1:])
				if totalStr != "*" {
					if v, parseErr := strconv.ParseInt(totalStr, 10, 64); parseErr == nil && v > 0 {
						size = v
					}
				}
			}
		}
	}

	log.Printf("[inferRemoteFileMetadata] Success for %s: mime=%s, size=%d", url, mimeType, size)
	return mimeType, size, nil
}

func mapMimeToType(mimeType, url string) string {
	mt := strings.ToLower(strings.TrimSpace(mimeType))

	// Проверяем MIME-type в первую очередь
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
		case strings.Contains(mt, "word") || strings.Contains(mt, "document"):
			return "Document"
		case strings.Contains(mt, "excel") || strings.Contains(mt, "spreadsheet"):
			return "Document"
		case strings.Contains(mt, "powerpoint") || strings.Contains(mt, "presentation"):
			return "Document"
		case strings.HasPrefix(mt, "text/"):
			return "Document"
		case strings.HasPrefix(mt, "application/zip") ||
			strings.Contains(mt, "compressed") ||
			strings.Contains(mt, "tar") ||
			strings.Contains(mt, "gzip") ||
			strings.Contains(mt, "7z") ||
			strings.Contains(mt, "rar"):
			return "Archive"
		}
	}

	// Fallback на расширение файла
	lower := strings.ToLower(url)

	// Documents
	if strings.HasSuffix(lower, ".pdf") ||
		strings.HasSuffix(lower, ".doc") ||
		strings.HasSuffix(lower, ".docx") ||
		strings.HasSuffix(lower, ".txt") ||
		strings.HasSuffix(lower, ".rtf") ||
		strings.HasSuffix(lower, ".odt") ||
		strings.HasSuffix(lower, ".xls") ||
		strings.HasSuffix(lower, ".xlsx") ||
		strings.HasSuffix(lower, ".ppt") ||
		strings.HasSuffix(lower, ".pptx") {
		return "Document"
	}

	// Archives
	if strings.HasSuffix(lower, ".zip") ||
		strings.HasSuffix(lower, ".tar") ||
		strings.HasSuffix(lower, ".gz") ||
		strings.HasSuffix(lower, ".7z") ||
		strings.HasSuffix(lower, ".rar") ||
		strings.HasSuffix(lower, ".bz2") {
		return "Archive"
	}

	// Images
	if strings.HasSuffix(lower, ".jpg") ||
		strings.HasSuffix(lower, ".jpeg") ||
		strings.HasSuffix(lower, ".png") ||
		strings.HasSuffix(lower, ".gif") ||
		strings.HasSuffix(lower, ".svg") ||
		strings.HasSuffix(lower, ".bmp") ||
		strings.HasSuffix(lower, ".webp") ||
		strings.HasSuffix(lower, ".ico") {
		return "Image"
	}

	// Videos
	if strings.HasSuffix(lower, ".mp4") ||
		strings.HasSuffix(lower, ".mov") ||
		strings.HasSuffix(lower, ".webm") ||
		strings.HasSuffix(lower, ".avi") ||
		strings.HasSuffix(lower, ".mkv") ||
		strings.HasSuffix(lower, ".flv") {
		return "Video"
	}

	// Audio
	if strings.HasSuffix(lower, ".mp3") ||
		strings.HasSuffix(lower, ".wav") ||
		strings.HasSuffix(lower, ".ogg") ||
		strings.HasSuffix(lower, ".flac") ||
		strings.HasSuffix(lower, ".aac") ||
		strings.HasSuffix(lower, ".m4a") {
		return "Audio"
	}

	return "Other"
}
