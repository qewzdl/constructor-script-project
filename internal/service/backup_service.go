package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/pkg/logger"

	"gorm.io/gorm"
)

const (
	backupSchemaVersion = "1"
	backupApplication   = "constructor-script"
)

var (
	ErrInvalidBackup = errors.New("invalid backup archive")
	ErrBackupVersion = errors.New("unsupported backup schema version")
)

type BackupService struct {
	db        *gorm.DB
	uploadDir string
	appName   string
}

type BackupSummary struct {
	SchemaVersion string    `json:"schema_version"`
	GeneratedAt   time.Time `json:"generated_at"`
	RestoredAt    time.Time `json:"restored_at,omitempty"`
	Application   string    `json:"application"`
	Users         int       `json:"users"`
	Categories    int       `json:"categories"`
	Tags          int       `json:"tags"`
	Posts         int       `json:"posts"`
	Pages         int       `json:"pages"`
	Comments      int       `json:"comments"`
	Settings      int       `json:"settings"`
	MenuItems     int       `json:"menu_items"`
	SocialLinks   int       `json:"social_links"`
	PostTags      int       `json:"post_tags"`
	Uploads       int       `json:"uploads"`
}

type BackupArchive struct {
	file     *os.File
	Filename string
	Summary  BackupSummary
}

type backupManifest struct {
	SchemaVersion string     `json:"schema_version"`
	GeneratedAt   time.Time  `json:"generated_at"`
	Application   string     `json:"application"`
	Uploads       []string   `json:"uploads"`
	Data          backupData `json:"data"`
}

type backupData struct {
	Users       []backupUser       `json:"users"`
	Categories  []backupCategory   `json:"categories"`
	Tags        []backupTag        `json:"tags"`
	Posts       []backupPost       `json:"posts"`
	Pages       []backupPage       `json:"pages"`
	Comments    []backupComment    `json:"comments"`
	Settings    []backupSetting    `json:"settings"`
	MenuItems   []backupMenuItem   `json:"menu_items"`
	SocialLinks []backupSocialLink `json:"social_links"`
	PostTags    []backupPostTag    `json:"post_tags"`
}

type backupUser struct {
	ID        uint       `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Password  string     `json:"password"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
}

type backupCategory struct {
	ID          uint       `json:"id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	Order       int        `json:"order"`
}

type backupTag struct {
	ID          uint       `json:"id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	UnusedSince *time.Time `json:"unused_since,omitempty"`
}

type backupPost struct {
	ID          uint                `json:"id"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	DeletedAt   *time.Time          `json:"deleted_at,omitempty"`
	Title       string              `json:"title"`
	Slug        string              `json:"slug"`
	Description string              `json:"description"`
	Content     string              `json:"content"`
	Excerpt     string              `json:"excerpt"`
	FeaturedImg string              `json:"featured_img"`
	Published   bool                `json:"published"`
	Views       int                 `json:"views"`
	Sections    models.PostSections `json:"sections"`
	Template    string              `json:"template"`
	AuthorID    uint                `json:"author_id"`
	CategoryID  uint                `json:"category_id"`
}

type backupPage struct {
	ID          uint                `json:"id"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	DeletedAt   *time.Time          `json:"deleted_at,omitempty"`
	Title       string              `json:"title"`
	Slug        string              `json:"slug"`
	Path        string              `json:"path"`
	Description string              `json:"description"`
	FeaturedImg string              `json:"featured_img"`
	Published   bool                `json:"published"`
	Content     string              `json:"content"`
	Sections    models.PostSections `json:"sections"`
	Template    string              `json:"template"`
	HideHeader  bool                `json:"hide_header"`
	Order       int                 `json:"order"`
}

type backupComment struct {
	ID        uint       `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	Content   string     `json:"content"`
	Approved  bool       `json:"approved"`
	PostID    uint       `json:"post_id"`
	AuthorID  uint       `json:"author_id"`
	ParentID  *uint      `json:"parent_id"`
}

type backupSetting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type backupMenuItem struct {
	ID        uint       `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	Title     string     `json:"title"`
	Label     string     `json:"label"`
	URL       string     `json:"url"`
	Location  string     `json:"location"`
	Order     int        `json:"order"`
}

type backupSocialLink struct {
	ID        uint       `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	Name      string     `json:"name"`
	URL       string     `json:"url"`
	Order     int        `json:"order"`
}

type backupPostTag struct {
	PostID uint `json:"post_id"`
	TagID  uint `json:"tag_id"`
}

type postTagRow struct {
	PostID uint
	TagID  uint
}

func NewBackupService(db *gorm.DB, uploadDir string) *BackupService {
	return &BackupService{
		db:        db,
		uploadDir: uploadDir,
		appName:   backupApplication,
	}
}

func (s *BackupService) CreateArchive(ctx context.Context) (*BackupArchive, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("backup service not configured")
	}

	manifest, err := s.buildManifest(ctx)
	if err != nil {
		return nil, err
	}

	tempFile, err := os.CreateTemp("", "constructor-backup-*.zip")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary archive: %w", err)
	}

	writer := zip.NewWriter(tempFile)

	if err := s.writeManifest(writer, manifest); err != nil {
		writer.Close()
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}

	if err := s.writeUploads(writer, manifest.Uploads); err != nil {
		writer.Close()
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}

	if err := writer.Close(); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, fmt.Errorf("failed to finalise backup archive: %w", err)
	}

	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, fmt.Errorf("failed to rewind backup archive: %w", err)
	}

	archiveName := fmt.Sprintf("backup-%s.zip", manifest.GeneratedAt.UTC().Format("20060102-150405"))

	summary := BackupSummary{
		SchemaVersion: manifest.SchemaVersion,
		GeneratedAt:   manifest.GeneratedAt,
		Application:   manifest.Application,
		Users:         len(manifest.Data.Users),
		Categories:    len(manifest.Data.Categories),
		Tags:          len(manifest.Data.Tags),
		Posts:         len(manifest.Data.Posts),
		Pages:         len(manifest.Data.Pages),
		Comments:      len(manifest.Data.Comments),
		Settings:      len(manifest.Data.Settings),
		MenuItems:     len(manifest.Data.MenuItems),
		SocialLinks:   len(manifest.Data.SocialLinks),
		PostTags:      len(manifest.Data.PostTags),
		Uploads:       len(manifest.Uploads),
	}

	return &BackupArchive{
		file:     tempFile,
		Filename: archiveName,
		Summary:  summary,
	}, nil
}

func (s *BackupService) RestoreArchive(ctx context.Context, reader io.Reader, size int64) (BackupSummary, error) {
	var summary BackupSummary

	if s == nil || s.db == nil {
		return summary, fmt.Errorf("backup service not configured")
	}

	spoolFile, err := os.CreateTemp("", "constructor-restore-*.zip")
	if err != nil {
		return summary, fmt.Errorf("failed to prepare temporary archive: %w", err)
	}
	defer func() {
		spoolFile.Close()
		os.Remove(spoolFile.Name())
	}()

	written, err := io.Copy(spoolFile, reader)
	if err != nil {
		return summary, fmt.Errorf("failed to read backup archive: %w", err)
	}
	if size > 0 && written != size {
		logger.Warn("Backup archive size mismatch", map[string]interface{}{
			"expected": size,
			"actual":   written,
		})
	}

	if _, err := spoolFile.Seek(0, io.SeekStart); err != nil {
		return summary, fmt.Errorf("failed to rewind archive: %w", err)
	}

	zipReader, err := zip.NewReader(spoolFile, written)
	if err != nil {
		return summary, fmt.Errorf("failed to read archive contents: %w", err)
	}

	manifest, err := s.loadManifest(zipReader)
	if err != nil {
		return summary, err
	}

	if manifest.SchemaVersion != backupSchemaVersion {
		return summary, ErrBackupVersion
	}

	tempUploadsDir, uploadsCount, err := s.extractUploads(zipReader)
	if err != nil {
		return summary, err
	}
	stagedUploads := false
	defer func() {
		if !stagedUploads {
			os.RemoveAll(tempUploadsDir)
		}
	}()

	tx := s.db.WithContext(ctx).Begin()
	if err := tx.Error; err != nil {
		return summary, fmt.Errorf("failed to start transaction: %w", err)
	}

	if err := s.resetDatabase(tx); err != nil {
		tx.Rollback()
		return summary, err
	}

	if err := s.restoreData(tx, manifest.Data); err != nil {
		tx.Rollback()
		return summary, err
	}

	if err := tx.Commit().Error; err != nil {
		return summary, fmt.Errorf("failed to commit restored data: %w", err)
	}

	backupDir, err := s.stageUploads(tempUploadsDir)
	if err != nil {
		// Attempt to rollback DB to previous backup if uploads fail is not practical here.
		return summary, err
	}
	stagedUploads = true

	summary = BackupSummary{
		SchemaVersion: manifest.SchemaVersion,
		GeneratedAt:   manifest.GeneratedAt,
		RestoredAt:    time.Now().UTC(),
		Application:   manifest.Application,
		Users:         len(manifest.Data.Users),
		Categories:    len(manifest.Data.Categories),
		Tags:          len(manifest.Data.Tags),
		Posts:         len(manifest.Data.Posts),
		Pages:         len(manifest.Data.Pages),
		Comments:      len(manifest.Data.Comments),
		Settings:      len(manifest.Data.Settings),
		MenuItems:     len(manifest.Data.MenuItems),
		SocialLinks:   len(manifest.Data.SocialLinks),
		PostTags:      len(manifest.Data.PostTags),
		Uploads:       uploadsCount,
	}

	if backupDir != "" {
		if err := os.RemoveAll(backupDir); err != nil {
			logger.Warn("Failed to remove upload backup after restore", map[string]interface{}{"path": backupDir, "error": err.Error()})
		}
	}

	return summary, nil
}

func (a *BackupArchive) File() *os.File {
	if a == nil {
		return nil
	}
	return a.file
}

func (a *BackupArchive) Reset() error {
	if a == nil || a.file == nil {
		return nil
	}
	_, err := a.file.Seek(0, io.SeekStart)
	return err
}

func (a *BackupArchive) Size() (int64, error) {
	if a == nil || a.file == nil {
		return 0, fmt.Errorf("archive not available")
	}
	info, err := a.file.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (a *BackupArchive) Close() error {
	if a == nil || a.file == nil {
		return nil
	}
	name := a.file.Name()
	err := a.file.Close()
	if removeErr := os.Remove(name); removeErr != nil {
		if err == nil {
			err = removeErr
		} else {
			logger.Warn("Failed to remove temporary backup archive", map[string]interface{}{"path": name, "error": removeErr.Error()})
		}
	}
	a.file = nil
	return err
}

func (s *BackupService) buildManifest(ctx context.Context) (backupManifest, error) {
	manifest := backupManifest{
		SchemaVersion: backupSchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		Application:   s.appName,
	}

	data, err := s.snapshotData(ctx)
	if err != nil {
		return manifest, err
	}
	manifest.Data = data

	uploads, err := s.listUploads()
	if err != nil {
		return manifest, err
	}
	manifest.Uploads = uploads

	return manifest, nil
}

func (s *BackupService) snapshotData(ctx context.Context) (backupData, error) {
	db := s.db.WithContext(ctx)
	result := backupData{}

	var users []models.User
	if err := db.Order("id ASC").Find(&users).Error; err != nil {
		return result, fmt.Errorf("failed to load users: %w", err)
	}
	result.Users = make([]backupUser, len(users))
	for i, user := range users {
		result.Users[i] = backupUser{
			ID:        user.ID,
			CreatedAt: user.CreatedAt.UTC(),
			UpdatedAt: user.UpdatedAt.UTC(),
			DeletedAt: deletedAtPtr(user.DeletedAt),
			Username:  user.Username,
			Email:     user.Email,
			Password:  user.Password,
			Role:      user.Role,
			Status:    user.Status,
		}
	}

	var categories []models.Category
	if err := db.Order("id ASC").Find(&categories).Error; err != nil {
		return result, fmt.Errorf("failed to load categories: %w", err)
	}
	result.Categories = make([]backupCategory, len(categories))
	for i, category := range categories {
		result.Categories[i] = backupCategory{
			ID:          category.ID,
			CreatedAt:   category.CreatedAt.UTC(),
			UpdatedAt:   category.UpdatedAt.UTC(),
			DeletedAt:   deletedAtPtr(category.DeletedAt),
			Name:        category.Name,
			Slug:        category.Slug,
			Description: category.Description,
			Order:       category.Order,
		}
	}

	var tags []models.Tag
	if err := db.Order("id ASC").Find(&tags).Error; err != nil {
		return result, fmt.Errorf("failed to load tags: %w", err)
	}
	result.Tags = make([]backupTag, len(tags))
	for i, tag := range tags {
		result.Tags[i] = backupTag{
			ID:          tag.ID,
			CreatedAt:   tag.CreatedAt.UTC(),
			UpdatedAt:   tag.UpdatedAt.UTC(),
			DeletedAt:   deletedAtPtr(tag.DeletedAt),
			Name:        tag.Name,
			Slug:        tag.Slug,
			UnusedSince: normalizeTimePtr(tag.UnusedSince),
		}
	}

	var posts []models.Post
	if err := db.Order("id ASC").Find(&posts).Error; err != nil {
		return result, fmt.Errorf("failed to load posts: %w", err)
	}
	result.Posts = make([]backupPost, len(posts))
	for i, post := range posts {
		result.Posts[i] = backupPost{
			ID:          post.ID,
			CreatedAt:   post.CreatedAt.UTC(),
			UpdatedAt:   post.UpdatedAt.UTC(),
			DeletedAt:   deletedAtPtr(post.DeletedAt),
			Title:       post.Title,
			Slug:        post.Slug,
			Description: post.Description,
			Content:     post.Content,
			Excerpt:     post.Excerpt,
			FeaturedImg: post.FeaturedImg,
			Published:   post.Published,
			Views:       post.Views,
			Sections:    post.Sections,
			Template:    post.Template,
			AuthorID:    post.AuthorID,
			CategoryID:  post.CategoryID,
		}
	}

	var pages []models.Page
	if err := db.Order("id ASC").Find(&pages).Error; err != nil {
		return result, fmt.Errorf("failed to load pages: %w", err)
	}
	result.Pages = make([]backupPage, len(pages))
	for i, page := range pages {
		result.Pages[i] = backupPage{
			ID:          page.ID,
			CreatedAt:   page.CreatedAt.UTC(),
			UpdatedAt:   page.UpdatedAt.UTC(),
			DeletedAt:   deletedAtPtr(page.DeletedAt),
			Title:       page.Title,
			Slug:        page.Slug,
			Path:        page.Path,
			Description: page.Description,
			FeaturedImg: page.FeaturedImg,
			Published:   page.Published,
			Content:     page.Content,
			Sections:    page.Sections,
			Template:    page.Template,
			HideHeader:  page.HideHeader,
			Order:       page.Order,
		}
	}

	var comments []models.Comment
	if err := db.Order("id ASC").Find(&comments).Error; err != nil {
		return result, fmt.Errorf("failed to load comments: %w", err)
	}
	result.Comments = make([]backupComment, len(comments))
	for i, comment := range comments {
		result.Comments[i] = backupComment{
			ID:        comment.ID,
			CreatedAt: comment.CreatedAt.UTC(),
			UpdatedAt: comment.UpdatedAt.UTC(),
			DeletedAt: deletedAtPtr(comment.DeletedAt),
			Content:   comment.Content,
			Approved:  comment.Approved,
			PostID:    comment.PostID,
			AuthorID:  comment.AuthorID,
			ParentID:  comment.ParentID,
		}
	}

	var settings []models.Setting
	if err := db.Order("key ASC").Find(&settings).Error; err != nil {
		return result, fmt.Errorf("failed to load settings: %w", err)
	}
	result.Settings = make([]backupSetting, len(settings))
	for i, setting := range settings {
		result.Settings[i] = backupSetting{
			Key:       setting.Key,
			Value:     setting.Value,
			CreatedAt: setting.CreatedAt.UTC(),
			UpdatedAt: setting.UpdatedAt.UTC(),
		}
	}

	var menuItems []models.MenuItem
	if err := db.Order("id ASC").Find(&menuItems).Error; err != nil {
		return result, fmt.Errorf("failed to load menu items: %w", err)
	}
	result.MenuItems = make([]backupMenuItem, len(menuItems))
	for i, item := range menuItems {
		result.MenuItems[i] = backupMenuItem{
			ID:        item.ID,
			CreatedAt: item.CreatedAt.UTC(),
			UpdatedAt: item.UpdatedAt.UTC(),
			DeletedAt: deletedAtPtr(item.DeletedAt),
			Title:     item.Title,
			Label:     item.Label,
			URL:       item.URL,
			Location:  item.Location,
			Order:     item.Order,
		}
	}

	var socialLinks []models.SocialLink
	if err := db.Order("id ASC").Find(&socialLinks).Error; err != nil {
		return result, fmt.Errorf("failed to load social links: %w", err)
	}
	result.SocialLinks = make([]backupSocialLink, len(socialLinks))
	for i, link := range socialLinks {
		result.SocialLinks[i] = backupSocialLink{
			ID:        link.ID,
			CreatedAt: link.CreatedAt.UTC(),
			UpdatedAt: link.UpdatedAt.UTC(),
			DeletedAt: deletedAtPtr(link.DeletedAt),
			Name:      link.Name,
			URL:       link.URL,
			Order:     link.Order,
		}
	}

	var postTagLinks []postTagRow
	if err := db.Table("post_tags").Order("post_id ASC, tag_id ASC").Find(&postTagLinks).Error; err != nil {
		return result, fmt.Errorf("failed to load post tags: %w", err)
	}
	result.PostTags = make([]backupPostTag, len(postTagLinks))
	for i, link := range postTagLinks {
		result.PostTags[i] = backupPostTag{PostID: link.PostID, TagID: link.TagID}
	}

	return result, nil
}

func (s *BackupService) listUploads() ([]string, error) {
	uploadDir := strings.TrimSpace(s.uploadDir)
	if uploadDir == "" {
		return nil, nil
	}

	info, err := os.Stat(uploadDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to inspect upload directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("upload path is not a directory")
	}

	files := make([]string, 0)
	err = filepath.WalkDir(uploadDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(uploadDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate uploads: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func (s *BackupService) writeManifest(writer *zip.Writer, manifest backupManifest) error {
	header := &zip.FileHeader{
		Name:   "manifest.json",
		Method: zip.Deflate,
	}
	header.SetModTime(manifest.GeneratedAt)
	w, err := writer.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create manifest entry: %w", err)
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("failed to encode manifest: %w", err)
	}
	return nil
}

func (s *BackupService) writeUploads(writer *zip.Writer, uploads []string) error {
	if len(uploads) == 0 {
		return nil
	}
	base := strings.TrimSpace(s.uploadDir)
	if base == "" {
		return nil
	}
	for _, rel := range uploads {
		absPath := filepath.Join(base, filepath.FromSlash(rel))
		info, err := os.Stat(absPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("failed to read upload file info: %w", err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to prepare archive header: %w", err)
		}
		header.Name = path.Join("uploads", rel)
		header.Method = zip.Deflate
		writerEntry, err := writer.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to create archive entry for upload: %w", err)
		}
		file, err := os.Open(absPath)
		if err != nil {
			return fmt.Errorf("failed to open upload file: %w", err)
		}
		if _, err := io.Copy(writerEntry, file); err != nil {
			file.Close()
			return fmt.Errorf("failed to write upload file to archive: %w", err)
		}
		file.Close()
	}
	return nil
}

func (s *BackupService) loadManifest(reader *zip.Reader) (backupManifest, error) {
	manifest := backupManifest{}
	var manifestFile *zip.File
	for _, file := range reader.File {
		if file.Name == "manifest.json" {
			manifestFile = file
			break
		}
	}
	if manifestFile == nil {
		return manifest, ErrInvalidBackup
	}
	rc, err := manifestFile.Open()
	if err != nil {
		return manifest, fmt.Errorf("failed to open manifest: %w", err)
	}
	defer rc.Close()

	decoder := json.NewDecoder(rc)
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("failed to decode manifest: %w", err)
	}
	if manifest.SchemaVersion == "" {
		return manifest, ErrInvalidBackup
	}
	return manifest, nil
}

func (s *BackupService) extractUploads(reader *zip.Reader) (string, int, error) {
	tempDir, err := os.MkdirTemp("", "constructor-uploads-*")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temporary uploads directory: %w", err)
	}

	count := 0
	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, "uploads/") {
			continue
		}
		relPath := strings.TrimPrefix(file.Name, "uploads/")
		relPath = path.Clean(relPath)
		if relPath == "." || relPath == "" {
			continue
		}
		segments := strings.Split(relPath, "/")
		invalid := false
		for _, segment := range segments {
			if segment == ".." || segment == "" {
				invalid = true
				break
			}
		}
		if invalid {
			return tempDir, count, fmt.Errorf("backup archive contains invalid upload path: %s", file.Name)
		}
		targetPath := filepath.Join(tempDir, filepath.FromSlash(relPath))

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return tempDir, count, fmt.Errorf("failed to create upload directory: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return tempDir, count, fmt.Errorf("failed to prepare upload destination: %w", err)
		}

		rc, err := file.Open()
		if err != nil {
			return tempDir, count, fmt.Errorf("failed to open upload from archive: %w", err)
		}

		mode := file.Mode()
		if mode == 0 {
			mode = 0o644
		}

		dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			rc.Close()
			return tempDir, count, fmt.Errorf("failed to create upload file: %w", err)
		}

		if _, err := io.Copy(dst, rc); err != nil {
			rc.Close()
			dst.Close()
			return tempDir, count, fmt.Errorf("failed to write upload file: %w", err)
		}

		rc.Close()
		dst.Close()
		count++
	}

	return tempDir, count, nil
}

func (s *BackupService) resetDatabase(tx *gorm.DB) error {
	stmt := "TRUNCATE TABLE post_tags, comments, posts, pages, categories, tags, menu_items, social_links, settings, users RESTART IDENTITY CASCADE"
	if err := tx.Exec(stmt).Error; err != nil {
		return fmt.Errorf("failed to reset database state: %w", err)
	}
	return nil
}

func (s *BackupService) restoreData(tx *gorm.DB, data backupData) error {
	if len(data.Users) > 0 {
		users := make([]models.User, len(data.Users))
		for i, item := range data.Users {
			users[i] = models.User{
				ID:        item.ID,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
				DeletedAt: deletedAtValue(item.DeletedAt),
				Username:  item.Username,
				Email:     item.Email,
				Password:  item.Password,
				Role:      item.Role,
				Status:    item.Status,
			}
		}
		if err := tx.Create(&users).Error; err != nil {
			return fmt.Errorf("failed to restore users: %w", err)
		}
	}

	if len(data.Categories) > 0 {
		categories := make([]models.Category, len(data.Categories))
		for i, item := range data.Categories {
			categories[i] = models.Category{
				ID:          item.ID,
				CreatedAt:   item.CreatedAt,
				UpdatedAt:   item.UpdatedAt,
				DeletedAt:   deletedAtValue(item.DeletedAt),
				Name:        item.Name,
				Slug:        item.Slug,
				Description: item.Description,
				Order:       item.Order,
			}
		}
		if err := tx.Create(&categories).Error; err != nil {
			return fmt.Errorf("failed to restore categories: %w", err)
		}
	}

	if len(data.Tags) > 0 {
		tags := make([]models.Tag, len(data.Tags))
		for i, item := range data.Tags {
			tags[i] = models.Tag{
				ID:          item.ID,
				CreatedAt:   item.CreatedAt,
				UpdatedAt:   item.UpdatedAt,
				DeletedAt:   deletedAtValue(item.DeletedAt),
				Name:        item.Name,
				Slug:        item.Slug,
				UnusedSince: normalizeTimePtr(item.UnusedSince),
			}
		}
		if err := tx.Create(&tags).Error; err != nil {
			return fmt.Errorf("failed to restore tags: %w", err)
		}
	}

	if len(data.Pages) > 0 {
		pages := make([]models.Page, len(data.Pages))
		for i, item := range data.Pages {
			pages[i] = models.Page{
				ID:          item.ID,
				CreatedAt:   item.CreatedAt,
				UpdatedAt:   item.UpdatedAt,
				DeletedAt:   deletedAtValue(item.DeletedAt),
				Title:       item.Title,
				Slug:        item.Slug,
				Path:        item.Path,
				Description: item.Description,
				FeaturedImg: item.FeaturedImg,
				Published:   item.Published,
				Content:     item.Content,
				Sections:    item.Sections,
				Template:    item.Template,
				HideHeader:  item.HideHeader,
				Order:       item.Order,
			}
		}
		if err := tx.Create(&pages).Error; err != nil {
			return fmt.Errorf("failed to restore pages: %w", err)
		}
	}

	if len(data.Posts) > 0 {
		posts := make([]models.Post, len(data.Posts))
		for i, item := range data.Posts {
			posts[i] = models.Post{
				ID:          item.ID,
				CreatedAt:   item.CreatedAt,
				UpdatedAt:   item.UpdatedAt,
				DeletedAt:   deletedAtValue(item.DeletedAt),
				Title:       item.Title,
				Slug:        item.Slug,
				Description: item.Description,
				Content:     item.Content,
				Excerpt:     item.Excerpt,
				FeaturedImg: item.FeaturedImg,
				Published:   item.Published,
				Views:       item.Views,
				Sections:    item.Sections,
				Template:    item.Template,
				AuthorID:    item.AuthorID,
				CategoryID:  item.CategoryID,
			}
		}
		if err := tx.Create(&posts).Error; err != nil {
			return fmt.Errorf("failed to restore posts: %w", err)
		}
	}

	if len(data.Comments) > 0 {
		comments := make([]models.Comment, len(data.Comments))
		for i, item := range data.Comments {
			comments[i] = models.Comment{
				ID:        item.ID,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
				DeletedAt: deletedAtValue(item.DeletedAt),
				Content:   item.Content,
				Approved:  item.Approved,
				PostID:    item.PostID,
				AuthorID:  item.AuthorID,
				ParentID:  item.ParentID,
			}
		}
		if err := tx.Create(&comments).Error; err != nil {
			return fmt.Errorf("failed to restore comments: %w", err)
		}
	}

	if len(data.MenuItems) > 0 {
		menuItems := make([]models.MenuItem, len(data.MenuItems))
		for i, item := range data.MenuItems {
			menuItems[i] = models.MenuItem{
				ID:        item.ID,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
				DeletedAt: deletedAtValue(item.DeletedAt),
				Title:     item.Title,
				Label:     item.Label,
				URL:       item.URL,
				Location:  item.Location,
				Order:     item.Order,
			}
		}
		if err := tx.Create(&menuItems).Error; err != nil {
			return fmt.Errorf("failed to restore menu items: %w", err)
		}
	}

	if len(data.SocialLinks) > 0 {
		links := make([]models.SocialLink, len(data.SocialLinks))
		for i, item := range data.SocialLinks {
			links[i] = models.SocialLink{
				ID:        item.ID,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
				DeletedAt: deletedAtValue(item.DeletedAt),
				Name:      item.Name,
				URL:       item.URL,
				Order:     item.Order,
			}
		}
		if err := tx.Create(&links).Error; err != nil {
			return fmt.Errorf("failed to restore social links: %w", err)
		}
	}

	if len(data.Settings) > 0 {
		settings := make([]models.Setting, len(data.Settings))
		for i, item := range data.Settings {
			settings[i] = models.Setting{
				Key:       item.Key,
				Value:     item.Value,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			}
		}
		if err := tx.Create(&settings).Error; err != nil {
			return fmt.Errorf("failed to restore settings: %w", err)
		}
	}

	if len(data.PostTags) > 0 {
		rows := make([]postTagRow, len(data.PostTags))
		for i, item := range data.PostTags {
			rows[i] = postTagRow{PostID: item.PostID, TagID: item.TagID}
		}
		if err := tx.Table("post_tags").Create(&rows).Error; err != nil {
			return fmt.Errorf("failed to restore post tags: %w", err)
		}
	}

	return nil
}

func (s *BackupService) stageUploads(tempDir string) (string, error) {
	base := strings.TrimSpace(s.uploadDir)
	if base == "" {
		return "", nil
	}

	backupDir := ""
	if _, err := os.Stat(base); err == nil {
		suffix := time.Now().UTC().Format("20060102-150405")
		backupDir = fmt.Sprintf("%s.bak-%s", base, suffix)
		if err := os.Rename(base, backupDir); err != nil {
			return "", fmt.Errorf("failed to backup existing uploads: %w", err)
		}
	}

	if tempDir == "" {
		if err := os.MkdirAll(base, 0o755); err != nil {
			s.rollbackUploads(backupDir)
			return "", fmt.Errorf("failed to prepare empty uploads directory: %w", err)
		}
		return backupDir, nil
	}

	if err := os.Rename(tempDir, base); err != nil {
		if err := copyDirectory(tempDir, base); err != nil {
			s.rollbackUploads(backupDir)
			return "", fmt.Errorf("failed to apply uploads: %w", err)
		}
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			logger.Warn("Failed to remove temporary uploads directory", map[string]interface{}{"path": tempDir, "error": removeErr.Error()})
		}
	}

	return backupDir, nil
}

func (s *BackupService) rollbackUploads(backupDir string) {
	base := strings.TrimSpace(s.uploadDir)
	if base != "" {
		os.RemoveAll(base)
	}
	if backupDir != "" {
		if err := os.Rename(backupDir, base); err != nil {
			logger.Warn("Failed to restore uploads from backup", map[string]interface{}{"backup": backupDir, "error": err.Error()})
		}
	}
}

func copyDirectory(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not supported in uploads")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			in.Close()
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			in.Close()
			return err
		}
		if err := out.Close(); err != nil {
			in.Close()
			return err
		}
		return in.Close()
	})
}

func deletedAtPtr(value gorm.DeletedAt) *time.Time {
	if value.Valid {
		t := value.Time.UTC()
		return &t
	}
	return nil
}

func deletedAtValue(value *time.Time) gorm.DeletedAt {
	if value == nil || value.IsZero() {
		return gorm.DeletedAt{}
	}
	return gorm.DeletedAt{Time: value.UTC(), Valid: true}
}

func normalizeTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := t.UTC()
	return &v
}
