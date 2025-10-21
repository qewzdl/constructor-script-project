package theme

import (
	"errors"
	"net/http"
	"sync/atomic"
)

var ErrThemeUnavailable = errors.New("theme assets unavailable")

type FileSystem struct {
	manager atomic.Pointer[Manager]
}

func NewFileSystem(manager *Manager) http.FileSystem {
	fs := &FileSystem{}
	fs.manager.Store(manager)
	return fs
}

func (f *FileSystem) Open(name string) (http.File, error) {
	manager := f.manager.Load()
	if manager == nil {
		return nil, ErrThemeUnavailable
	}

	theme := manager.Active()
	if theme == nil {
		return nil, ErrThemeUnavailable
	}

	dir := http.Dir(theme.StaticDir)
	return dir.Open(name)
}
