package service

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
)

type PackageService struct {
	packageRepo repository.CoursePackageRepository
	topicRepo   repository.CourseTopicRepository
	videoRepo   repository.CourseVideoRepository
}

func NewPackageService(
	packageRepo repository.CoursePackageRepository,
	topicRepo repository.CourseTopicRepository,
	videoRepo repository.CourseVideoRepository,
) *PackageService {
	return &PackageService{
		packageRepo: packageRepo,
		topicRepo:   topicRepo,
		videoRepo:   videoRepo,
	}
}

func (s *PackageService) SetRepositories(
	packageRepo repository.CoursePackageRepository,
	topicRepo repository.CourseTopicRepository,
	videoRepo repository.CourseVideoRepository,
) {
	if s == nil {
		return
	}
	s.packageRepo = packageRepo
	s.topicRepo = topicRepo
	s.videoRepo = videoRepo
}

func (s *PackageService) Create(req models.CreateCoursePackageRequest) (*models.CoursePackage, error) {
	if s == nil || s.packageRepo == nil {
		return nil, errors.New("course package repository is not configured")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, errors.New("package title is required")
	}
	if req.PriceCents < 0 {
		return nil, errors.New("package price must be zero or positive")
	}

	pkg := models.CoursePackage{
		Title:       title,
		Description: strings.TrimSpace(req.Description),
		PriceCents:  req.PriceCents,
		ImageURL:    strings.TrimSpace(req.ImageURL),
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
		return nil, errors.New("package title is required")
	}
	if req.PriceCents < 0 {
		return nil, errors.New("package price must be zero or positive")
	}

	pkg.Title = title
	pkg.Description = strings.TrimSpace(req.Description)
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

	packages := []models.CoursePackage{*pkg}
	if err := s.populateTopics(packages); err != nil {
		return nil, err
	}

	result := packages[0]
	return &result, nil
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
		return fmt.Errorf("one or more topics do not exist")
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

	if err := s.populateTopicVideos(topicMap); err != nil {
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

func (s *PackageService) populateTopicVideos(topics map[uint]*models.CourseTopic) error {
	if len(topics) == 0 {
		return nil
	}
	if s.topicRepo == nil || s.videoRepo == nil {
		return errors.New("course topic repository is not configured")
	}

	topicIDs := make([]uint, 0, len(topics))
	for id, topic := range topics {
		topic.Videos = []models.CourseVideo{}
		topicIDs = append(topicIDs, id)
	}

	linksByTopic, err := s.topicRepo.ListVideoLinks(topicIDs)
	if err != nil {
		return err
	}

	if len(linksByTopic) == 0 {
		return nil
	}

	videoIDSet := make(map[uint]struct{})
	for _, links := range linksByTopic {
		for _, link := range links {
			videoIDSet[link.VideoID] = struct{}{}
		}
	}

	if len(videoIDSet) == 0 {
		return nil
	}

	videoIDs := make([]uint, 0, len(videoIDSet))
	for id := range videoIDSet {
		videoIDs = append(videoIDs, id)
	}

	videos, err := s.videoRepo.GetByIDs(videoIDs)
	if err != nil {
		return err
	}

	videoMap := make(map[uint]models.CourseVideo, len(videos))
	for _, video := range videos {
		videoMap[video.ID] = video
	}

	for topicID, links := range linksByTopic {
		topic, ok := topics[topicID]
		if !ok {
			continue
		}
		ordered := make([]models.CourseVideo, 0, len(links))
		for _, link := range links {
			if video, exists := videoMap[link.VideoID]; exists {
				ordered = append(ordered, video)
			}
		}
		topic.Videos = ordered
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
