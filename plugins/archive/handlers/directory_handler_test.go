package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"constructor-script-backend/internal/models"
	archiveservice "constructor-script-backend/plugins/archive/service"
)

type stubArchiveDirectoryRepository struct {
	listAllResult      []models.ArchiveDirectory
	listAllError       error
	listAllCalls       int
	listByParentResult []models.ArchiveDirectory
	listByParentError  error
	listByParentCalls  int
}

func (s *stubArchiveDirectoryRepository) Create(directory *models.ArchiveDirectory) error {
	return errors.New("not implemented")
}

func (s *stubArchiveDirectoryRepository) Update(directory *models.ArchiveDirectory) error {
	return errors.New("not implemented")
}

func (s *stubArchiveDirectoryRepository) Delete(id uint) error {
	return errors.New("not implemented")
}

func (s *stubArchiveDirectoryRepository) GetByID(id uint) (*models.ArchiveDirectory, error) {
	return nil, errors.New("not implemented")
}

func (s *stubArchiveDirectoryRepository) GetByPath(path string) (*models.ArchiveDirectory, error) {
	return nil, errors.New("not implemented")
}

func (s *stubArchiveDirectoryRepository) ListAll(includeUnpublished bool) ([]models.ArchiveDirectory, error) {
	s.listAllCalls++
	if s.listAllError != nil {
		return nil, s.listAllError
	}
	result := make([]models.ArchiveDirectory, len(s.listAllResult))
	copy(result, s.listAllResult)
	return result, nil
}

func (s *stubArchiveDirectoryRepository) ListByParent(parentID *uint, includeUnpublished bool) ([]models.ArchiveDirectory, error) {
	s.listByParentCalls++
	if s.listByParentError != nil {
		return nil, s.listByParentError
	}
	result := make([]models.ArchiveDirectory, len(s.listByParentResult))
	copy(result, s.listByParentResult)
	return result, nil
}

func (s *stubArchiveDirectoryRepository) ExistsBySlugAndParent(slug string, parentID *uint, excludeID *uint) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *stubArchiveDirectoryRepository) ExistsByPath(path string, excludeID *uint) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *stubArchiveDirectoryRepository) ListDescendants(path string) ([]models.ArchiveDirectory, error) {
	return nil, errors.New("not implemented")
}

func (s *stubArchiveDirectoryRepository) CountChildren(id uint) (int64, error) {
	return 0, errors.New("not implemented")
}

type stubArchiveFileRepository struct {
	listAllResult []models.ArchiveFile
	listAllError  error
}

func (s *stubArchiveFileRepository) Create(file *models.ArchiveFile) error {
	return errors.New("not implemented")
}

func (s *stubArchiveFileRepository) Update(file *models.ArchiveFile) error {
	return errors.New("not implemented")
}

func (s *stubArchiveFileRepository) Delete(id uint) error {
	return errors.New("not implemented")
}

func (s *stubArchiveFileRepository) GetByID(id uint) (*models.ArchiveFile, error) {
	return nil, errors.New("not implemented")
}

func (s *stubArchiveFileRepository) GetByPath(path string) (*models.ArchiveFile, error) {
	return nil, errors.New("not implemented")
}

func (s *stubArchiveFileRepository) ListAll(includeUnpublished bool) ([]models.ArchiveFile, error) {
	if s.listAllError != nil {
		return nil, s.listAllError
	}
	result := make([]models.ArchiveFile, len(s.listAllResult))
	copy(result, s.listAllResult)
	return result, nil
}

func (s *stubArchiveFileRepository) ListByDirectory(directoryID uint, includeUnpublished bool) ([]models.ArchiveFile, error) {
	return nil, errors.New("not implemented")
}

func (s *stubArchiveFileRepository) ExistsBySlug(directoryID uint, slug string, excludeID *uint) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *stubArchiveFileRepository) ListByDirectoryPath(path string) ([]models.ArchiveFile, error) {
	return nil, errors.New("not implemented")
}

func (s *stubArchiveFileRepository) CountByDirectory(directoryID uint) (int64, error) {
	return 0, errors.New("not implemented")
}

func TestParseTreeFlagVariants(t *testing.T) {
	cases := map[string]bool{
		"1":      true,
		"true":   true,
		"TRUE":   true,
		"yes":    true,
		"y":      true,
		"on":     true,
		"1:1":    true,
		"true:1": true,
		"0":      false,
		"false":  false,
		"off":    false,
		"":       false,
		":":      false,
		"maybe":  false,
	}

	for input, expected := range cases {
		if actual := parseTreeFlag(input); actual != expected {
			t.Fatalf("parseTreeFlag(%q) = %v, expected %v", input, actual, expected)
		}
	}
}

func TestDirectoryHandlerListTreeParameterWithDelimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubArchiveDirectoryRepository{
		listAllResult: []models.ArchiveDirectory{{
			ID:       1,
			Name:     "Root",
			Slug:     "root",
			Path:     "root",
			Children: nil,
		}},
	}
	fileRepo := &stubArchiveFileRepository{}
	service := archiveservice.NewDirectoryService(repo, fileRepo, nil)
	handler := NewDirectoryHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/?tree=1:1", nil)
	c.Request = req

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var payload struct {
		Directories []models.ArchiveDirectory `json:"directories"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(payload.Directories) != len(repo.listAllResult) {
		t.Fatalf("expected %d directories, got %d", len(repo.listAllResult), len(payload.Directories))
	}

	if repo.listAllCalls == 0 {
		t.Fatalf("expected ListAll to be called at least once")
	}

	if repo.listByParentCalls != 0 {
		t.Fatalf("expected ListByParent not to be called, but was called %d times", repo.listByParentCalls)
	}
}

func TestDirectoryHandlerListDefaultsToParentListing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubArchiveDirectoryRepository{
		listByParentResult: []models.ArchiveDirectory{{
			ID:   42,
			Name: "Child",
			Slug: "child",
			Path: "root/child",
		}},
	}
	fileRepo := &stubArchiveFileRepository{}
	service := archiveservice.NewDirectoryService(repo, fileRepo, nil)
	handler := NewDirectoryHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request = req

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if repo.listAllCalls != 0 {
		t.Fatalf("expected ListAll not to be called, but was called %d times", repo.listAllCalls)
	}

	if repo.listByParentCalls == 0 {
		t.Fatalf("expected ListByParent to be called")
	}
}
