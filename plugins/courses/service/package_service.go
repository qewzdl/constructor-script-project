package service

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
)

type PackageService struct {
	packageRepo repository.CoursePackageRepository
	topicRepo   repository.CourseTopicRepository
	videoRepo   repository.CourseVideoRepository
	testRepo    repository.CourseTestRepository
	accessRepo  repository.CoursePackageAccessRepository
	userRepo    repository.UserRepository
}

func NewPackageService(
	packageRepo repository.CoursePackageRepository,
	topicRepo repository.CourseTopicRepository,
	videoRepo repository.CourseVideoRepository,
	testRepo repository.CourseTestRepository,
	accessRepo repository.CoursePackageAccessRepository,
	userRepo repository.UserRepository,
) *PackageService {
	return &PackageService{
		packageRepo: packageRepo,
		topicRepo:   topicRepo,
		videoRepo:   videoRepo,
		testRepo:    testRepo,
		accessRepo:  accessRepo,
		userRepo:    userRepo,
	}
}

func (s *PackageService) SetRepositories(
	packageRepo repository.CoursePackageRepository,
	topicRepo repository.CourseTopicRepository,
	videoRepo repository.CourseVideoRepository,
	testRepo repository.CourseTestRepository,
	accessRepo repository.CoursePackageAccessRepository,
	userRepo repository.UserRepository,
) {
	if s == nil {
		return
	}
	s.packageRepo = packageRepo
	s.topicRepo = topicRepo
	s.videoRepo = videoRepo
	s.testRepo = testRepo
	s.accessRepo = accessRepo
	s.userRepo = userRepo
}

func (s *PackageService) Create(req models.CreateCoursePackageRequest) (*models.CoursePackage, error) {
	if s == nil || s.packageRepo == nil {
		return nil, errors.New("course package repository is not configured")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, newValidationError("package title is required")
	}
	if req.PriceCents < 0 {
		return nil, newValidationError("package price must be zero or positive")
	}

	slug := normalizeSlug(req.Slug)
	if slug == "" {
		return nil, newValidationError("package slug is required")
	}

	if existing, err := s.packageRepo.GetBySlug(slug); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else if existing != nil {
		return nil, newValidationError("package slug is already in use")
	}

	pkg := models.CoursePackage{
		Title:           title,
		Slug:            slug,
		Summary:         strings.TrimSpace(req.Summary),
		Description:     strings.TrimSpace(req.Description),
		MetaTitle:       strings.TrimSpace(req.MetaTitle),
		MetaDescription: strings.TrimSpace(req.MetaDescription),
		PriceCents:      req.PriceCents,
		ImageURL:        strings.TrimSpace(req.ImageURL),
	}

	if err := s.packageRepo.Create(&pkg); err != nil {
		return nil, err
	}

	if len(req.TopicIDs) > 0 {
		if err := s.assignTopics(pkg.ID, req.TopicIDs); err != nil {
			return nil, err
		}
	}

	return s.GetByID(pkg.ID)
}

func (s *PackageService) Update(id uint, req models.UpdateCoursePackageRequest) (*models.CoursePackage, error) {
	if s == nil || s.packageRepo == nil {
		return nil, errors.New("course package repository is not configured")
	}

	pkg, err := s.packageRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, newValidationError("package title is required")
	}
	if req.PriceCents < 0 {
		return nil, newValidationError("package price must be zero or positive")
	}

	slug := normalizeSlug(req.Slug)
	if slug == "" {
		return nil, newValidationError("package slug is required")
	}

	if existing, err := s.packageRepo.GetBySlug(slug); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else if existing != nil && existing.ID != pkg.ID {
		return nil, newValidationError("package slug is already in use")
	}

	pkg.Title = title
	pkg.Slug = slug
	pkg.Summary = strings.TrimSpace(req.Summary)
	pkg.Description = strings.TrimSpace(req.Description)
	pkg.MetaTitle = strings.TrimSpace(req.MetaTitle)
	pkg.MetaDescription = strings.TrimSpace(req.MetaDescription)
	pkg.PriceCents = req.PriceCents
	pkg.ImageURL = strings.TrimSpace(req.ImageURL)

	if err := s.packageRepo.Update(pkg); err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

func (s *PackageService) Delete(id uint) error {
	if s == nil || s.packageRepo == nil {
		return errors.New("course package repository is not configured")
	}
	return s.packageRepo.Delete(id)
}

func (s *PackageService) GetByID(id uint) (*models.CoursePackage, error) {
	if s == nil || s.packageRepo == nil {
		return nil, errors.New("course package repository is not configured")
	}

	pkg, err := s.packageRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return s.preparePackage(pkg)
}

func (s *PackageService) GetBySlug(slug string) (*models.CoursePackage, error) {
	if s == nil || s.packageRepo == nil {
		return nil, errors.New("course package repository is not configured")
	}

	normalized := normalizeSlug(slug)
	if normalized == "" {
		return nil, gorm.ErrRecordNotFound
	}

	pkg, err := s.packageRepo.GetBySlug(normalized)
	if err != nil {
		return nil, err
	}

	return s.preparePackage(pkg)
}

func (s *PackageService) GetByIdentifier(identifier string) (*models.CoursePackage, error) {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return nil, gorm.ErrRecordNotFound
	}

	if id, err := strconv.ParseUint(trimmed, 10, 64); err == nil && id > 0 {
		return s.GetByID(uint(id))
	}

	return s.GetBySlug(trimmed)
}

func (s *PackageService) List() ([]models.CoursePackage, error) {
	if s == nil || s.packageRepo == nil {
		return nil, errors.New("course package repository is not configured")
	}

	packages, err := s.packageRepo.List()
	if err != nil {
		return nil, err
	}

	if err := s.populateTopics(packages); err != nil {
		return nil, err
	}

	return packages, nil
}

func (s *PackageService) UpdateTopics(packageID uint, topicIDs []uint) (*models.CoursePackage, error) {
	if s == nil || s.packageRepo == nil {
		return nil, errors.New("course package repository is not configured")
	}

	exists, err := s.packageRepo.Exists(packageID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, gorm.ErrRecordNotFound
	}

	if err := s.assignTopics(packageID, topicIDs); err != nil {
		return nil, err
	}

	return s.GetByID(packageID)
}

func (s *PackageService) GrantToUser(packageID uint, req models.GrantCoursePackageRequest, grantedBy uint) (*models.CoursePackageAccess, error) {
	if s == nil || s.packageRepo == nil || s.accessRepo == nil || s.userRepo == nil {
		return nil, errors.New("course package service is not fully configured")
	}
	if packageID == 0 {
		return nil, newValidationError("package id is required")
	}
	if req.UserID == 0 {
		return nil, newValidationError("user id is required")
	}

	exists, err := s.packageRepo.Exists(packageID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, gorm.ErrRecordNotFound
	}

	if _, err := s.userRepo.GetByID(req.UserID); err != nil {
		return nil, err
	}

	var current *models.CoursePackageAccess
	if !req.ExpiresAt.Set || grantedBy == 0 {
		existing, err := s.accessRepo.GetByUserAndPackage(req.UserID, packageID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if err == nil {
			current = existing
		}
	}

	access := models.CoursePackageAccess{
		PackageID: packageID,
		UserID:    req.UserID,
	}

	if req.ExpiresAt.Set {
		access.ExpiresAt = normalizeTimePointer(req.ExpiresAt.Pointer())
	} else if current != nil {
		access.ExpiresAt = normalizeTimePointer(current.ExpiresAt)
	}

	if grantedBy > 0 {
		id := grantedBy
		access.GrantedBy = &id
	} else if current != nil {
		access.GrantedBy = cloneUintPointer(current.GrantedBy)
	}

	if err := s.accessRepo.Upsert(&access); err != nil {
		return nil, err
	}

	return s.accessRepo.GetByUserAndPackage(req.UserID, packageID)
}

func (s *PackageService) ListForUser(userID uint) ([]models.UserCoursePackage, error) {
	result := make([]models.UserCoursePackage, 0)
	if s == nil || s.packageRepo == nil || s.accessRepo == nil {
		return result, errors.New("course package service is not fully configured")
	}
	if userID == 0 {
		return result, newValidationError("user id is required")
	}

	accesses, err := s.accessRepo.ListActiveByUser(userID)
	if err != nil {
		return nil, err
	}
	if len(accesses) == 0 {
		return result, nil
	}

	packageIDs := make([]uint, 0, len(accesses))
	for _, access := range accesses {
		packageIDs = append(packageIDs, access.PackageID)
	}

	uniqueIDs := uniqueOrdered(packageIDs)
	packages, err := s.packageRepo.GetByIDs(uniqueIDs)
	if err != nil {
		return nil, err
	}
	if len(packages) > 0 {
		if err := s.populateTopics(packages); err != nil {
			return nil, err
		}
	}

	packageMap := make(map[uint]models.CoursePackage, len(packages))
	for _, pkg := range packages {
		packageMap[pkg.ID] = pkg
	}

	for _, access := range accesses {
		if pkg, ok := packageMap[access.PackageID]; ok {
			result = append(result, models.UserCoursePackage{
				Package: pkg,
				Access:  access,
			})
		}
	}

	return result, nil
}

func (s *PackageService) preparePackage(pkg *models.CoursePackage) (*models.CoursePackage, error) {
	if s == nil {
		return nil, errors.New("course package service is not configured")
	}
	if pkg == nil {
		return nil, errors.New("course package is required")
	}

	packages := []models.CoursePackage{*pkg}
	if err := s.populateTopics(packages); err != nil {
		return nil, err
	}

	result := packages[0]
	return &result, nil
}

func (s *PackageService) buildUserCourse(pkg *models.CoursePackage, userID uint) (*models.UserCoursePackage, error) {
	if s == nil || s.accessRepo == nil {
		return nil, errors.New("course package service is not fully configured")
	}
	if pkg == nil {
		return nil, errors.New("course package is required")
	}

	access, err := s.accessRepo.GetByUserAndPackage(userID, pkg.ID)
	if err != nil {
		return nil, err
	}
	if access == nil {
		return nil, gorm.ErrRecordNotFound
	}
	if access.ExpiresAt != nil {
		if access.ExpiresAt.Before(time.Now()) {
			return nil, gorm.ErrRecordNotFound
		}
	}

	prepared, err := s.preparePackage(pkg)
	if err != nil {
		return nil, err
	}

	result := models.UserCoursePackage{
		Package: *prepared,
		Access:  *access,
	}
	return &result, nil
}

func (s *PackageService) GetForUser(packageID, userID uint) (*models.UserCoursePackage, error) {
	if s == nil || s.packageRepo == nil || s.accessRepo == nil {
		return nil, errors.New("course package service is not fully configured")
	}
	if packageID == 0 {
		return nil, newValidationError("package id is required")
	}
	if userID == 0 {
		return nil, newValidationError("user id is required")
	}

	pkg, err := s.packageRepo.GetByID(packageID)
	if err != nil {
		return nil, err
	}

	return s.buildUserCourse(pkg, userID)
}

func (s *PackageService) GetForUserByIdentifier(identifier string, userID uint) (*models.UserCoursePackage, error) {
	if s == nil || s.packageRepo == nil || s.accessRepo == nil {
		return nil, errors.New("course package service is not fully configured")
	}
	if userID == 0 {
		return nil, newValidationError("user id is required")
	}

	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return nil, gorm.ErrRecordNotFound
	}

	if id, err := strconv.ParseUint(trimmed, 10, 64); err == nil && id > 0 {
		return s.GetForUser(uint(id), userID)
	}

	normalized := normalizeSlug(trimmed)
	if normalized == "" {
		return nil, gorm.ErrRecordNotFound
	}

	pkg, err := s.packageRepo.GetBySlug(normalized)
	if err != nil {
		return nil, err
	}

	return s.buildUserCourse(pkg, userID)
}

func (s *PackageService) assignTopics(packageID uint, topicIDs []uint) error {
	if s.topicRepo == nil {
		return errors.New("course topic repository is not configured")
	}
	if len(topicIDs) == 0 {
		return s.packageRepo.SetTopics(packageID, nil)
	}

	unique := uniqueOrdered(topicIDs)
	topics, err := s.topicRepo.GetByIDs(unique)
	if err != nil {
		return err
	}
	if len(topics) != len(unique) {
		return newValidationError("one or more topics do not exist")
	}

	return s.packageRepo.SetTopics(packageID, unique)
}

func (s *PackageService) populateTopics(packages []models.CoursePackage) error {
	if len(packages) == 0 {
		return nil
	}
	if s.packageRepo == nil || s.topicRepo == nil {
		return errors.New("course package repository is not configured")
	}

	packageIDs := make([]uint, 0, len(packages))
	for i := range packages {
		packages[i].Topics = []models.CourseTopic{}
		packageIDs = append(packageIDs, packages[i].ID)
	}

	linksByPackage, err := s.packageRepo.ListTopicLinks(packageIDs)
	if err != nil {
		return err
	}

	if len(linksByPackage) == 0 {
		return nil
	}

	topicIDSet := make(map[uint]struct{})
	for _, links := range linksByPackage {
		for _, link := range links {
			topicIDSet[link.TopicID] = struct{}{}
		}
	}

	if len(topicIDSet) == 0 {
		return nil
	}

	topicIDs := make([]uint, 0, len(topicIDSet))
	for id := range topicIDSet {
		topicIDs = append(topicIDs, id)
	}

	topics, err := s.topicRepo.GetByIDs(topicIDs)
	if err != nil {
		return err
	}

	topicMap := make(map[uint]*models.CourseTopic, len(topics))
	for i := range topics {
		topics[i].Videos = []models.CourseVideo{}
		topicMap[topics[i].ID] = &topics[i]
	}

	if err := s.populateTopicSteps(topicMap); err != nil {
		return err
	}

	packageMap := make(map[uint]*models.CoursePackage, len(packages))
	for i := range packages {
		pkg := &packages[i]
		packageMap[pkg.ID] = pkg
	}

	for packageID, links := range linksByPackage {
		pkg, ok := packageMap[packageID]
		if !ok {
			continue
		}
		ordered := make([]models.CourseTopic, 0, len(links))
		for _, link := range links {
			if topic, exists := topicMap[link.TopicID]; exists {
				ordered = append(ordered, *topic)
			}
		}
		pkg.Topics = ordered
	}

	return nil
}

func (s *PackageService) populateTopicSteps(topics map[uint]*models.CourseTopic) error {
	if len(topics) == 0 {
		return nil
	}
	if s.topicRepo == nil {
		return errors.New("course topic repository is not configured")
	}

	topicIDs := make([]uint, 0, len(topics))
	for id, topic := range topics {
		topic.Videos = []models.CourseVideo{}
		topic.Steps = []models.CourseTopicStep{}
		topicIDs = append(topicIDs, id)
	}

	linksByTopic, err := s.topicRepo.ListStepLinks(topicIDs)
	if err != nil {
		return err
	}
	if len(linksByTopic) == 0 {
		return nil
	}

	videoIDSet := make(map[uint]struct{})
	testIDSet := make(map[uint]struct{})
	for _, links := range linksByTopic {
		for _, link := range links {
			if link.StepType == models.CourseTopicStepTypeVideo && link.VideoID != nil {
				videoIDSet[*link.VideoID] = struct{}{}
			}
			if link.StepType == models.CourseTopicStepTypeTest && link.TestID != nil {
				testIDSet[*link.TestID] = struct{}{}
			}
		}
	}

	videoMap := make(map[uint]models.CourseVideo, len(videoIDSet))
	if len(videoIDSet) > 0 {
		if s.videoRepo == nil {
			return errors.New("course video repository is not configured")
		}
		ids := make([]uint, 0, len(videoIDSet))
		for id := range videoIDSet {
			ids = append(ids, id)
		}
		videos, err := s.videoRepo.GetByIDs(ids)
		if err != nil {
			return err
		}
		for _, video := range videos {
			videoMap[video.ID] = video
		}
	}

	testMap := make(map[uint]models.CourseTest, len(testIDSet))
	if len(testIDSet) > 0 {
		if s.testRepo == nil {
			return errors.New("course test repository is not configured")
		}
		ids := make([]uint, 0, len(testIDSet))
		for id := range testIDSet {
			ids = append(ids, id)
		}
		tests, err := s.testRepo.GetByIDs(ids)
		if err != nil {
			return err
		}
		structures, err := s.testRepo.ListStructure(ids)
		if err != nil {
			return err
		}
		for i := range tests {
			test := tests[i]
			if questions, ok := structures[test.ID]; ok {
				test.Questions = questions
			} else {
				test.Questions = []models.CourseTestQuestion{}
			}
			testMap[test.ID] = test
		}
	}

	for topicID, links := range linksByTopic {
		topic, ok := topics[topicID]
		if !ok {
			continue
		}
		ordered := make([]models.CourseTopicStep, 0, len(links))
		for _, link := range links {
			step := link
			step.Video = nil
			step.Test = nil
			if link.StepType == models.CourseTopicStepTypeVideo && link.VideoID != nil {
				if video, exists := videoMap[*link.VideoID]; exists {
					videoCopy := video
					step.Video = &videoCopy
					topic.Videos = append(topic.Videos, videoCopy)
				}
			}
			if link.StepType == models.CourseTopicStepTypeTest && link.TestID != nil {
				if test, exists := testMap[*link.TestID]; exists {
					testCopy := test
					step.Test = &testCopy
				}
			}
			ordered = append(ordered, step)
		}
		topic.Steps = ordered
	}

	return nil
}

func uniqueOrdered(values []uint) []uint {
	if len(values) == 0 {
		return []uint{}
	}
	seen := make(map[uint]struct{}, len(values))
	ordered := make([]uint, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ordered = append(ordered, value)
	}
	return ordered
}

func normalizeTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := value.UTC().Truncate(time.Second)
	return &normalized
}

func cloneUintPointer(value *uint) *uint {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
