package theme

import (
	"errors"
	"io/fs"
	"net/http"
	"strings"
	"sync/atomic"
)

var ErrThemeUnavailable = errors.New("theme assets unavailable")

type FileSystem struct {
	manager atomic.Pointer[Manager]
	siteFS  http.FileSystem
}

func NewFileSystem(manager *Manager, siteDir string) http.FileSystem {
	fs := &FileSystem{}
	fs.manager.Store(manager)
	if cleaned := strings.TrimSpace(siteDir); cleaned != "" {
		fs.siteFS = http.Dir(cleaned)
	}
	return fs
}

func (f *FileSystem) Open(name string) (http.File, error) {
	if f.siteFS != nil {
		file, err := f.siteFS.Open(name)
		if err == nil {
			return file, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

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
