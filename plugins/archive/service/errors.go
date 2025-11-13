package service

import "errors"

var (
	ErrDirectoryNotFound = errors.New("directory not found")
	ErrFileNotFound      = errors.New("file not found")
	ErrInvalidParent     = errors.New("invalid parent directory")
	ErrDirectoryNotEmpty = errors.New("directory is not empty")
	ErrSlugConflict      = errors.New("slug already in use")
)
