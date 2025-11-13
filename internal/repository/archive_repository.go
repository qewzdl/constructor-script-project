package repository

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"constructor-script-backend/internal/models"
)

type ArchiveDirectoryRepository interface {
	Create(directory *models.ArchiveDirectory) error
	Update(directory *models.ArchiveDirectory) error
	Delete(id uint) error
	GetByID(id uint) (*models.ArchiveDirectory, error)
	GetByPath(path string) (*models.ArchiveDirectory, error)
	ListAll(includeUnpublished bool) ([]models.ArchiveDirectory, error)
	ListByParent(parentID *uint, includeUnpublished bool) ([]models.ArchiveDirectory, error)
	ExistsBySlugAndParent(slug string, parentID *uint, excludeID *uint) (bool, error)
	ExistsByPath(path string, excludeID *uint) (bool, error)
	ListDescendants(path string) ([]models.ArchiveDirectory, error)
	CountChildren(id uint) (int64, error)
}

type ArchiveFileRepository interface {
	Create(file *models.ArchiveFile) error
	Update(file *models.ArchiveFile) error
	Delete(id uint) error
	GetByID(id uint) (*models.ArchiveFile, error)
	GetByPath(path string) (*models.ArchiveFile, error)
	ListAll(includeUnpublished bool) ([]models.ArchiveFile, error)
	ListByDirectory(directoryID uint, includeUnpublished bool) ([]models.ArchiveFile, error)
	ExistsBySlug(directoryID uint, slug string, excludeID *uint) (bool, error)
	ListByDirectoryPath(path string) ([]models.ArchiveFile, error)
	CountByDirectory(directoryID uint) (int64, error)
}

type archiveDirectoryRepository struct {
	db *gorm.DB
}

type archiveFileRepository struct {
	db *gorm.DB
}

func NewArchiveDirectoryRepository(db *gorm.DB) ArchiveDirectoryRepository {
	return &archiveDirectoryRepository{db: db}
}

func NewArchiveFileRepository(db *gorm.DB) ArchiveFileRepository {
	return &archiveFileRepository{db: db}
}

func normalizePath(path string) string {
	return strings.TrimSpace(strings.ToLower(path))
}

func (r *archiveDirectoryRepository) Create(directory *models.ArchiveDirectory) error {
	if directory == nil {
		return gorm.ErrInvalidData
	}
	directory.Path = normalizePath(directory.Path)
	return r.db.Create(directory).Error
}

func (r *archiveDirectoryRepository) Update(directory *models.ArchiveDirectory) error {
	if directory == nil {
		return gorm.ErrInvalidData
	}
	directory.Path = normalizePath(directory.Path)
	return r.db.Save(directory).Error
}

func (r *archiveDirectoryRepository) Delete(id uint) error {
	return r.db.Unscoped().Delete(&models.ArchiveDirectory{}, id).Error
}

func (r *archiveDirectoryRepository) GetByID(id uint) (*models.ArchiveDirectory, error) {
	var directory models.ArchiveDirectory
	err := r.db.First(&directory, id).Error
	if err != nil {
		return nil, err
	}
	return &directory, nil
}

func (r *archiveDirectoryRepository) GetByPath(path string) (*models.ArchiveDirectory, error) {
	normalized := normalizePath(path)
	var directory models.ArchiveDirectory
	err := r.db.Where("LOWER(path) = ?", normalized).First(&directory).Error
	if err != nil {
		return nil, err
	}
	return &directory, nil
}

func (r *archiveDirectoryRepository) ListAll(includeUnpublished bool) ([]models.ArchiveDirectory, error) {
	var directories []models.ArchiveDirectory
	query := r.db.Model(&models.ArchiveDirectory{})
	if !includeUnpublished {
		query = query.Where("published = ?", true)
	}
	err := query.
		Order("COALESCE(parent_id, 0) ASC").
		Order(clause.OrderByColumn{Column: clause.Column{Name: "order"}}).
		Order("LOWER(name) ASC").
		Find(&directories).Error
	return directories, err
}

func (r *archiveDirectoryRepository) ListByParent(parentID *uint, includeUnpublished bool) ([]models.ArchiveDirectory, error) {
	var directories []models.ArchiveDirectory
	query := r.db.Model(&models.ArchiveDirectory{})
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}
	if !includeUnpublished {
		query = query.Where("published = ?", true)
	}
	err := query.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "order"}}).
		Order("LOWER(name) ASC").
		Find(&directories).Error
	return directories, err
}

func (r *archiveDirectoryRepository) ExistsBySlugAndParent(slug string, parentID *uint, excludeID *uint) (bool, error) {
	normalized := strings.TrimSpace(strings.ToLower(slug))
	if normalized == "" {
		return false, nil
	}
	query := r.db.Model(&models.ArchiveDirectory{}).Where("slug = ?", normalized)
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}
	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *archiveDirectoryRepository) ExistsByPath(path string, excludeID *uint) (bool, error) {
	normalized := normalizePath(path)
	query := r.db.Model(&models.ArchiveDirectory{}).Where("LOWER(path) = ?", normalized)
	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *archiveDirectoryRepository) ListDescendants(path string) ([]models.ArchiveDirectory, error) {
	normalized := normalizePath(path)
	if normalized == "" {
		return nil, nil
	}
	likePattern := normalized + "/%"
	var directories []models.ArchiveDirectory
	err := r.db.Where("LOWER(path) LIKE ?", likePattern).Order("LENGTH(path) ASC").Find(&directories).Error
	return directories, err
}

func (r *archiveDirectoryRepository) CountChildren(id uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.ArchiveDirectory{}).Where("parent_id = ?", id).Count(&count).Error
	return count, err
}

func (r *archiveFileRepository) Create(file *models.ArchiveFile) error {
	if file == nil {
		return gorm.ErrInvalidData
	}
	file.Path = normalizePath(file.Path)
	return r.db.Create(file).Error
}

func (r *archiveFileRepository) Update(file *models.ArchiveFile) error {
	if file == nil {
		return gorm.ErrInvalidData
	}
	file.Path = normalizePath(file.Path)
	return r.db.Save(file).Error
}

func (r *archiveFileRepository) Delete(id uint) error {
	return r.db.Unscoped().Delete(&models.ArchiveFile{}, id).Error
}

func (r *archiveFileRepository) GetByID(id uint) (*models.ArchiveFile, error) {
	var file models.ArchiveFile
	err := r.db.First(&file, id).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *archiveFileRepository) GetByPath(path string) (*models.ArchiveFile, error) {
	normalized := normalizePath(path)
	var file models.ArchiveFile
	err := r.db.Where("LOWER(path) = ?", normalized).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *archiveFileRepository) ListAll(includeUnpublished bool) ([]models.ArchiveFile, error) {
	var files []models.ArchiveFile
	query := r.db.Model(&models.ArchiveFile{})
	if !includeUnpublished {
		query = query.Where("published = ?", true)
	}
	err := query.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "directory_id"}}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "order"}}).
		Order("LOWER(name) ASC").
		Find(&files).Error
	return files, err
}

func (r *archiveFileRepository) ListByDirectory(directoryID uint, includeUnpublished bool) ([]models.ArchiveFile, error) {
	var files []models.ArchiveFile
	query := r.db.Where("directory_id = ?", directoryID)
	if !includeUnpublished {
		query = query.Where("published = ?", true)
	}
	err := query.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "order"}}).
		Order("LOWER(name) ASC").
		Find(&files).Error
	return files, err
}

func (r *archiveFileRepository) ExistsBySlug(directoryID uint, slug string, excludeID *uint) (bool, error) {
	normalized := strings.TrimSpace(strings.ToLower(slug))
	if normalized == "" {
		return false, nil
	}
	query := r.db.Model(&models.ArchiveFile{}).
		Where("directory_id = ?", directoryID).
		Where("slug = ?", normalized)
	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *archiveFileRepository) ListByDirectoryPath(path string) ([]models.ArchiveFile, error) {
	normalized := normalizePath(path)
	if normalized == "" {
		return nil, nil
	}
	likePattern := normalized + "/%"
	var files []models.ArchiveFile
	err := r.db.Where("LOWER(path) LIKE ?", likePattern).Find(&files).Error
	return files, err
}

func (r *archiveFileRepository) CountByDirectory(directoryID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.ArchiveFile{}).Where("directory_id = ?", directoryID).Count(&count).Error
	return count, err
}
