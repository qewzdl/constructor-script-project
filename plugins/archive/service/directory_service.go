package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/cache"
	"constructor-script-backend/pkg/utils"
)

const (
	cacheKeyTreeAll       = "archive:tree:all"
	cacheKeyTreePublished = "archive:tree:published"
)

type DirectoryService struct {
	directoryRepo repository.ArchiveDirectoryRepository
	fileRepo      repository.ArchiveFileRepository
	cache         *cache.Cache
}

func NewDirectoryService(directoryRepo repository.ArchiveDirectoryRepository, fileRepo repository.ArchiveFileRepository, cacheService *cache.Cache) *DirectoryService {
	return &DirectoryService{
		directoryRepo: directoryRepo,
		fileRepo:      fileRepo,
		cache:         cacheService,
	}
}

func clearDirectoryDescription(directory *models.ArchiveDirectory) {
	if directory == nil {
		return
	}
	directory.Description = ""
}

func clearDirectoryDescriptions(directories []models.ArchiveDirectory) {
	for i := range directories {
		directories[i].Description = ""
		if len(directories[i].Children) > 0 {
			clearDirectoryDescriptions(directories[i].Children)
		}
	}
}

func (s *DirectoryService) Create(req models.CreateArchiveDirectoryRequest) (*models.ArchiveDirectory, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("directory name is required")
	}

	var parent *models.ArchiveDirectory
	parentID := req.ParentID.Pointer()
	if parentID != nil {
		fetched, err := s.directoryRepo.GetByID(*parentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrInvalidParent
			}
			return nil, err
		}
		parent = fetched
	}

	slug := sanitizeDirectorySlug(req.Slug)
	if slug == "" {
		slug = utils.GenerateSlug(name)
	}
	if slug == "" {
		slug = fmt.Sprintf("directory-%d", time.Now().UnixNano())
	}

	uniqueSlug, err := s.ensureDirectorySlug(slug, parentID, nil)
	if err != nil {
		return nil, err
	}

	path := buildDirectoryPath(parent, uniqueSlug)
	exists, err := s.directoryRepo.ExistsByPath(path, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrSlugConflict
	}

	directory := &models.ArchiveDirectory{
		Name:        name,
		Slug:        uniqueSlug,
		Path:        path,
		Description: "",
		Order:       req.Order,
		Published:   req.Published,
	}
	if parentID != nil {
		directory.ParentID = parentID
	}

	if err := s.directoryRepo.Create(directory); err != nil {
		return nil, err
	}

	s.invalidateTreeCache()
	return directory, nil
}

func (s *DirectoryService) Update(id uint, req models.UpdateArchiveDirectoryRequest) (*models.ArchiveDirectory, error) {
	directory, err := s.directoryRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDirectoryNotFound
		}
		return nil, err
	}

	originalPath := directory.Path
	originalParent := directory.ParentID

	var parent *models.ArchiveDirectory
	parentChanged := false

	if req.ParentID.Set {
		if req.ParentID.Value == nil {
			directory.ParentID = nil
			parentChanged = originalParent != nil
		} else {
			newParentID := *req.ParentID.Value
			if newParentID == directory.ID {
				return nil, ErrInvalidParent
			}
			fetched, fetchErr := s.directoryRepo.GetByID(newParentID)
			if fetchErr != nil {
				if errors.Is(fetchErr, gorm.ErrRecordNotFound) {
					return nil, ErrInvalidParent
				}
				return nil, fetchErr
			}
			if strings.EqualFold(fetched.Path, directory.Path) || strings.HasPrefix(fetched.Path+"/", directory.Path+"/") {
				return nil, ErrInvalidParent
			}
			parent = fetched
			directory.ParentID = &newParentID
			parentChanged = originalParent == nil || *originalParent != newParentID
		}
	} else if directory.ParentID != nil {
		fetched, _ := s.directoryRepo.GetByID(*directory.ParentID)
		parent = fetched
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("directory name cannot be empty")
		}
		directory.Name = name
	}

	slug := directory.Slug
	slugChanged := false

	if req.Slug != nil {
		provided := sanitizeDirectorySlug(*req.Slug)
		if provided == "" && directory.Name != "" {
			provided = utils.GenerateSlug(directory.Name)
		}
		if provided == "" {
			provided = fmt.Sprintf("directory-%d", directory.ID)
		}
		if provided != "" && !strings.EqualFold(provided, directory.Slug) {
			slug = provided
			slugChanged = true
		}
	}

	if slug == "" {
		generated := utils.GenerateSlug(directory.Name)
		if generated == "" {
			generated = fmt.Sprintf("directory-%d", directory.ID)
		}
		slug = generated
		if !strings.EqualFold(slug, directory.Slug) {
			slugChanged = true
		}
	}

	parentID := directory.ParentID
	if parent == nil && parentID != nil {
		fetched, fetchErr := s.directoryRepo.GetByID(*parentID)
		if fetchErr != nil {
			if errors.Is(fetchErr, gorm.ErrRecordNotFound) {
				return nil, ErrInvalidParent
			}
			return nil, fetchErr
		}
		parent = fetched
	}

	if slugChanged || parentChanged {
		uniqueSlug, err := s.ensureDirectorySlug(slug, parentID, &directory.ID)
		if err != nil {
			return nil, err
		}
		slug = uniqueSlug
	}

	directory.Slug = slug
	directory.Path = buildDirectoryPath(parent, slug)

	directory.Description = ""
	if req.Published != nil {
		directory.Published = *req.Published
	}
	if req.Order != nil {
		directory.Order = *req.Order
	}

	if err := s.directoryRepo.Update(directory); err != nil {
		return nil, err
	}

	if !strings.EqualFold(originalPath, directory.Path) {
		if err := s.relocateDescendants(originalPath, directory.Path); err != nil {
			return nil, err
		}
	}

	if err := s.realignDirectoryFiles(directory); err != nil {
		return nil, err
	}

	s.invalidateTreeCache()
	return directory, nil
}

func (s *DirectoryService) Delete(id uint) error {
	directory, err := s.directoryRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDirectoryNotFound
		}
		return err
	}

	if count, err := s.directoryRepo.CountChildren(id); err != nil {
		return err
	} else if count > 0 {
		return ErrDirectoryNotEmpty
	}

	if count, err := s.fileRepo.CountByDirectory(id); err != nil {
		return err
	} else if count > 0 {
		return ErrDirectoryNotEmpty
	}

	if err := s.directoryRepo.Delete(directory.ID); err != nil {
		return err
	}

	s.invalidateTreeCache()
	return nil
}

func (s *DirectoryService) GetByID(id uint, includeUnpublished bool) (*models.ArchiveDirectory, error) {
	directory, err := s.directoryRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDirectoryNotFound
		}
		return nil, err
	}
	if !includeUnpublished && !directory.Published {
		return nil, ErrDirectoryNotFound
	}
	clearDirectoryDescription(directory)
	return directory, nil
}

func (s *DirectoryService) GetByPath(path string, includeUnpublished bool) (*models.ArchiveDirectory, error) {
	directory, err := s.directoryRepo.GetByPath(path)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDirectoryNotFound
		}
		return nil, err
	}
	if !includeUnpublished && !directory.Published {
		return nil, ErrDirectoryNotFound
	}
	clearDirectoryDescription(directory)
	return directory, nil
}

func (s *DirectoryService) ListByParent(parentID *uint, includeUnpublished bool) ([]models.ArchiveDirectory, error) {
	directories, err := s.directoryRepo.ListByParent(parentID, includeUnpublished)
	if err != nil {
		return nil, err
	}
	clearDirectoryDescriptions(directories)
	return directories, nil
}

func (s *DirectoryService) ListTree(includeUnpublished bool) ([]models.ArchiveDirectory, error) {
	cacheKey := cacheKeyTreeAll
	if !includeUnpublished {
		cacheKey = cacheKeyTreePublished
	}

	if s.cache != nil {
		var cached []models.ArchiveDirectory
		if err := s.cache.Get(cacheKey, &cached); err == nil {
			clearDirectoryDescriptions(cached)
			return cached, nil
		}
	}

	tree, err := s.buildTree(includeUnpublished)
	if err != nil {
		return nil, err
	}
	clearDirectoryDescriptions(tree)

	if s.cache != nil {
		_ = s.cache.Set(cacheKey, tree, 30*time.Minute)
	}

	return tree, nil
}

func (s *DirectoryService) ListPublishedTree() ([]models.ArchiveDirectory, error) {
	return s.ListTree(false)
}

func (s *DirectoryService) BuildBreadcrumbs(path string, includeUnpublished bool) ([]models.ArchiveBreadcrumb, error) {
	normalized := strings.TrimSpace(strings.ToLower(path))
	if normalized == "" {
		return nil, nil
	}

	parts := strings.Split(normalized, "/")
	breadcrumbs := make([]models.ArchiveBreadcrumb, 0, len(parts))
	current := ""

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if current == "" {
			current = part
		} else {
			current = current + "/" + part
		}

		directory, err := s.directoryRepo.GetByPath(current)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrDirectoryNotFound
			}
			return nil, err
		}
		if !includeUnpublished && !directory.Published {
			return nil, ErrDirectoryNotFound
		}

		breadcrumbs = append(breadcrumbs, models.ArchiveBreadcrumb{
			Name: strings.TrimSpace(directory.Name),
			Path: directory.Path,
		})
	}

	return breadcrumbs, nil
}

func (s *DirectoryService) InvalidateTreeCache() {
	s.invalidateTreeCache()
}

func (s *DirectoryService) buildTree(includeUnpublished bool) ([]models.ArchiveDirectory, error) {
	directories, err := s.directoryRepo.ListAll(includeUnpublished)
	if err != nil {
		return nil, err
	}
	clearDirectoryDescriptions(directories)
	files, err := s.fileRepo.ListAll(includeUnpublished)
	if err != nil {
		return nil, err
	}

	nodes := make(map[uint]*models.ArchiveDirectory, len(directories))
	childRefs := make(map[uint][]*models.ArchiveDirectory)
	roots := make([]*models.ArchiveDirectory, 0)

	for i := range directories {
		dir := &directories[i]
		dir.Children = nil
		dir.Files = nil
		nodes[dir.ID] = dir
		if dir.ParentID != nil {
			childRefs[*dir.ParentID] = append(childRefs[*dir.ParentID], dir)
		} else {
			roots = append(roots, dir)
		}
	}

	fileRefs := make(map[uint][]models.ArchiveFile)
	for i := range files {
		file := files[i]
		fileRefs[file.DirectoryID] = append(fileRefs[file.DirectoryID], file)
	}

	sortNodes := func(entries []*models.ArchiveDirectory) {
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].Order == entries[j].Order {
				return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
			}
			return entries[i].Order < entries[j].Order
		})
	}

	sortFiles := func(entries []models.ArchiveFile) []models.ArchiveFile {
		if len(entries) == 0 {
			return nil
		}
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].Order == entries[j].Order {
				return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
			}
			return entries[i].Order < entries[j].Order
		})
		result := make([]models.ArchiveFile, len(entries))
		copy(result, entries)
		return result
	}

	var build func(dir *models.ArchiveDirectory)
	build = func(dir *models.ArchiveDirectory) {
		children := childRefs[dir.ID]
		if len(children) > 0 {
			sortNodes(children)
			dir.Children = make([]models.ArchiveDirectory, 0, len(children))
			for _, child := range children {
				build(child)
				dir.Children = append(dir.Children, *child)
			}
		} else {
			dir.Children = nil
		}

		dir.Files = sortFiles(fileRefs[dir.ID])
	}

	if len(roots) > 0 {
		sortNodes(roots)
	}

	result := make([]models.ArchiveDirectory, 0, len(roots))
	for _, root := range roots {
		build(root)
		result = append(result, *root)
	}

	return result, nil
}

func (s *DirectoryService) ensureDirectorySlug(base string, parentID *uint, excludeID *uint) (string, error) {
	candidate := strings.TrimSpace(strings.ToLower(base))
	if candidate == "" {
		candidate = "directory"
	}
	baseSlug := candidate
	suffix := 1

	for {
		exists, err := s.directoryRepo.ExistsBySlugAndParent(candidate, parentID, excludeID)
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

func (s *DirectoryService) relocateDescendants(oldPath, newPath string) error {
	oldNormalized := strings.TrimSpace(strings.ToLower(oldPath))
	newNormalized := strings.TrimSpace(strings.ToLower(newPath))
	if oldNormalized == "" || oldNormalized == newNormalized {
		return nil
	}

	descendants, err := s.directoryRepo.ListDescendants(oldNormalized)
	if err != nil {
		return err
	}

	if len(descendants) == 0 {
		return nil
	}

	oldPrefix := oldNormalized + "/"
	newPrefix := newNormalized + "/"

	for i := range descendants {
		child := descendants[i]
		trimmed := strings.TrimPrefix(child.Path, oldPrefix)
		child.Path = newPrefix + trimmed
		if err := s.directoryRepo.Update(&child); err != nil {
			return err
		}
		if err := s.realignDirectoryFiles(&child); err != nil {
			return err
		}
	}

	return nil
}

func (s *DirectoryService) realignDirectoryFiles(directory *models.ArchiveDirectory) error {
	if directory == nil {
		return nil
	}
	files, err := s.fileRepo.ListByDirectory(directory.ID, true)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	base := strings.TrimSpace(strings.ToLower(directory.Path))
	for i := range files {
		file := files[i]
		file.Path = buildFilePath(base, file.Slug)
		if err := s.fileRepo.Update(&file); err != nil {
			return err
		}
	}
	return nil
}

func (s *DirectoryService) invalidateTreeCache() {
	if s.cache == nil {
		return
	}
	_ = s.cache.Delete(cacheKeyTreeAll)
	_ = s.cache.Delete(cacheKeyTreePublished)
}

func sanitizeDirectorySlug(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	slug := utils.GenerateSlug(value)
	if slug == "" {
		return ""
	}
	return slug
}

func buildDirectoryPath(parent *models.ArchiveDirectory, slug string) string {
	base := strings.TrimSpace(strings.ToLower(slug))
	if parent == nil || strings.TrimSpace(parent.Path) == "" {
		return base
	}
	parentPath := strings.TrimSpace(strings.ToLower(parent.Path))
	return parentPath + "/" + base
}

func buildFilePath(directoryPath, slug string) string {
	dir := strings.Trim(strings.TrimSpace(strings.ToLower(directoryPath)), "/")
	fileSlug := strings.TrimSpace(strings.ToLower(slug))
	if dir == "" {
		return fileSlug
	}
	return dir + "/" + fileSlug
}
