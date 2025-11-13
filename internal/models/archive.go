package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type ArchiveDirectory struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string            `gorm:"not null" json:"name"`
	Slug        string            `gorm:"not null;index:idx_archive_directories_parent_slug,priority:2" json:"slug"`
	Path        string            `gorm:"not null;uniqueIndex" json:"path"`
	Description string            `json:"description"`
	Order       int               `gorm:"default:0" json:"order"`
	Published   bool              `gorm:"default:true" json:"published"`
	ParentID    *uint             `gorm:"index;index:idx_archive_directories_parent_slug,priority:1" json:"parent_id"`
	Parent      *ArchiveDirectory `gorm:"foreignKey:ParentID" json:"parent,omitempty"`

	Children []ArchiveDirectory `gorm:"-" json:"children,omitempty"`
	Files    []ArchiveFile      `gorm:"-" json:"files,omitempty"`
}

type ArchiveFile struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	DirectoryID uint             `gorm:"index;not null" json:"directory_id"`
	Directory   ArchiveDirectory `gorm:"foreignKey:DirectoryID" json:"directory,omitempty"`

	Name        string `gorm:"not null" json:"name"`
	Slug        string `gorm:"not null;index:idx_archive_files_directory_slug,priority:2" json:"slug"`
	Path        string `gorm:"not null;uniqueIndex" json:"path"`
	Description string `json:"description"`
	FileURL     string `gorm:"not null" json:"file_url"`
	PreviewURL  string `json:"preview_url"`
	MimeType    string `json:"mime_type"`
	FileType    string `json:"file_type"`
	FileSize    int64  `gorm:"default:0" json:"file_size"`
	Order       int    `gorm:"default:0" json:"order"`
	Published   bool   `gorm:"default:true" json:"published"`
}

type ArchiveDirectorySummary struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

type CreateArchiveDirectoryRequest struct {
	Name        string       `json:"name" binding:"required"`
	Slug        string       `json:"slug"`
	Description string       `json:"description"`
	ParentID    OptionalUint `json:"parent_id"`
	Published   bool         `json:"published"`
	Order       int          `json:"order"`
}

type UpdateArchiveDirectoryRequest struct {
	Name        *string      `json:"name"`
	Slug        *string      `json:"slug"`
	Description *string      `json:"description"`
	ParentID    OptionalUint `json:"parent_id"`
	Published   *bool        `json:"published"`
	Order       *int         `json:"order"`
}

type CreateArchiveFileRequest struct {
	DirectoryID uint   `json:"directory_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	FileURL     string `json:"file_url" binding:"required"`
	PreviewURL  string `json:"preview_url"`
	MimeType    string `json:"mime_type"`
	FileType    string `json:"file_type"`
	FileSize    *int64 `json:"file_size"`
	Published   bool   `json:"published"`
	Order       int    `json:"order"`
}

type UpdateArchiveFileRequest struct {
	DirectoryID OptionalUint `json:"directory_id"`
	Name        *string      `json:"name"`
	Slug        *string      `json:"slug"`
	Description *string      `json:"description"`
	FileURL     *string      `json:"file_url"`
	PreviewURL  *string      `json:"preview_url"`
	MimeType    *string      `json:"mime_type"`
	FileType    *string      `json:"file_type"`
	FileSize    *int64       `json:"file_size"`
	Published   *bool        `json:"published"`
	Order       *int         `json:"order"`
}

type ArchiveBreadcrumb struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (d *ArchiveDirectory) NormalizedPath() string {
	if d == nil {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(d.Path))
}

func (f *ArchiveFile) NormalizedPath() string {
	if f == nil {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(f.Path))
}
