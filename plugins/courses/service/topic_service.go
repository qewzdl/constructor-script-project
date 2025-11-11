package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
)

type TopicService struct {
	topicRepo repository.CourseTopicRepository
	videoRepo repository.CourseVideoRepository
	testRepo  repository.CourseTestRepository
}

func NewTopicService(
	topicRepo repository.CourseTopicRepository,
	videoRepo repository.CourseVideoRepository,
	testRepo repository.CourseTestRepository,
) *TopicService {
	return &TopicService{
		topicRepo: topicRepo,
		videoRepo: videoRepo,
		testRepo:  testRepo,
	}
}

func (s *TopicService) SetRepositories(
	topicRepo repository.CourseTopicRepository,
	videoRepo repository.CourseVideoRepository,
	testRepo repository.CourseTestRepository,
) {
	if s == nil {
		return
	}
	s.topicRepo = topicRepo
	s.videoRepo = videoRepo
	s.testRepo = testRepo
}

func (s *TopicService) Create(req models.CreateCourseTopicRequest) (*models.CourseTopic, error) {
	if s == nil || s.topicRepo == nil {
		return nil, errors.New("course topic repository is not configured")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, newValidationError("topic title is required")
	}

	slug := normalizeSlug(req.Slug)
	if slug == "" {
		return nil, newValidationError("topic slug is required")
	}

	if existing, err := s.topicRepo.GetBySlug(slug); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else if existing != nil {
		return nil, newValidationError("topic slug is already in use")
	}

	topic := models.CourseTopic{
		Title:           title,
		Slug:            slug,
		Summary:         strings.TrimSpace(req.Summary),
		Description:     strings.TrimSpace(req.Description),
		MetaTitle:       strings.TrimSpace(req.MetaTitle),
		MetaDescription: strings.TrimSpace(req.MetaDescription),
	}

	if err := s.topicRepo.Create(&topic); err != nil {
		return nil, err
	}

	if len(req.VideoIDs) > 0 {
		if err := s.assignVideos(topic.ID, req.VideoIDs); err != nil {
			return nil, err
		}
	}

	return s.GetByID(topic.ID)
}

func (s *TopicService) Update(id uint, req models.UpdateCourseTopicRequest) (*models.CourseTopic, error) {
	if s == nil || s.topicRepo == nil {
		return nil, errors.New("course topic repository is not configured")
	}

	topic, err := s.topicRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, newValidationError("topic title is required")
	}

	slug := normalizeSlug(req.Slug)
	if slug == "" {
		return nil, newValidationError("topic slug is required")
	}

	if existing, err := s.topicRepo.GetBySlug(slug); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else if existing != nil && existing.ID != topic.ID {
		return nil, newValidationError("topic slug is already in use")
	}

	topic.Title = title
	topic.Slug = slug
	topic.Summary = strings.TrimSpace(req.Summary)
	topic.Description = strings.TrimSpace(req.Description)
	topic.MetaTitle = strings.TrimSpace(req.MetaTitle)
	topic.MetaDescription = strings.TrimSpace(req.MetaDescription)

	if err := s.topicRepo.Update(topic); err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

func (s *TopicService) Delete(id uint) error {
	if s == nil || s.topicRepo == nil {
		return errors.New("course topic repository is not configured")
	}
	return s.topicRepo.Delete(id)
}

func (s *TopicService) GetByID(id uint) (*models.CourseTopic, error) {
	if s == nil || s.topicRepo == nil {
		return nil, errors.New("course topic repository is not configured")
	}

	topic, err := s.topicRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return s.prepareTopic(topic)
}

func (s *TopicService) GetBySlug(slug string) (*models.CourseTopic, error) {
	if s == nil || s.topicRepo == nil {
		return nil, errors.New("course topic repository is not configured")
	}

	normalized := normalizeSlug(slug)
	if normalized == "" {
		return nil, gorm.ErrRecordNotFound
	}

	topic, err := s.topicRepo.GetBySlug(normalized)
	if err != nil {
		return nil, err
	}

	return s.prepareTopic(topic)
}

func (s *TopicService) GetByIdentifier(identifier string) (*models.CourseTopic, error) {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return nil, gorm.ErrRecordNotFound
	}

	if id, err := strconv.ParseUint(trimmed, 10, 64); err == nil && id > 0 {
		return s.GetByID(uint(id))
	}

	return s.GetBySlug(trimmed)
}

func (s *TopicService) List() ([]models.CourseTopic, error) {
	if s == nil || s.topicRepo == nil {
		return nil, errors.New("course topic repository is not configured")
	}

	topics, err := s.topicRepo.List()
	if err != nil {
		return nil, err
	}

	if err := s.populateSteps(topics); err != nil {
		return nil, err
	}

	return topics, nil
}

func (s *TopicService) UpdateVideos(topicID uint, videoIDs []uint) (*models.CourseTopic, error) {
	refs := make([]models.CourseTopicStepReference, 0, len(videoIDs))
	for _, id := range videoIDs {
		refs = append(refs, models.CourseTopicStepReference{Type: models.CourseTopicStepTypeVideo, ID: id})
	}
	return s.UpdateSteps(topicID, refs)
}

func (s *TopicService) UpdateSteps(topicID uint, refs []models.CourseTopicStepReference) (*models.CourseTopic, error) {
	if s == nil || s.topicRepo == nil {
		return nil, errors.New("course topic repository is not configured")
	}

	exists, err := s.topicRepo.Exists(topicID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, gorm.ErrRecordNotFound
	}

	steps, err := s.buildSteps(refs)
	if err != nil {
		return nil, err
	}

	if err := s.topicRepo.SetSteps(topicID, steps); err != nil {
		return nil, err
	}

	return s.GetByID(topicID)
}

func (s *TopicService) prepareTopic(topic *models.CourseTopic) (*models.CourseTopic, error) {
	if topic == nil {
		return nil, errors.New("topic is required")
	}

	topics := []models.CourseTopic{*topic}
	if err := s.populateSteps(topics); err != nil {
		return nil, err
	}

	result := topics[0]
	return &result, nil
}

func (s *TopicService) buildSteps(refs []models.CourseTopicStepReference) ([]models.CourseTopicStep, error) {
	steps := make([]models.CourseTopicStep, 0, len(refs))
	if len(refs) == 0 {
		return steps, nil
	}

	seen := make(map[string]struct{}, len(refs))
	normalized := make([]models.CourseTopicStepReference, 0, len(refs))
	videoIDSet := make(map[uint]struct{})
	testIDSet := make(map[uint]struct{})

	for _, ref := range refs {
		stepType := strings.ToLower(strings.TrimSpace(ref.Type))
		switch stepType {
		case models.CourseTopicStepTypeVideo:
			videoIDSet[ref.ID] = struct{}{}
		case models.CourseTopicStepTypeTest:
			testIDSet[ref.ID] = struct{}{}
		default:
			return nil, newValidationError("invalid step type: %s", ref.Type)
		}

		key := fmt.Sprintf("%s:%d", stepType, ref.ID)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, models.CourseTopicStepReference{Type: stepType, ID: ref.ID})
	}

	if len(videoIDSet) > 0 {
		if s.videoRepo == nil {
			return nil, errors.New("course video repository is not configured")
		}
		ids := make([]uint, 0, len(videoIDSet))
		for id := range videoIDSet {
			ids = append(ids, id)
		}
		videos, err := s.videoRepo.GetByIDs(ids)
		if err != nil {
			return nil, err
		}
		if len(videos) != len(videoIDSet) {
			return nil, newValidationError("one or more videos do not exist")
		}
	}

	if len(testIDSet) > 0 {
		if s.testRepo == nil {
			return nil, errors.New("course test repository is not configured")
		}
		ids := make([]uint, 0, len(testIDSet))
		for id := range testIDSet {
			ids = append(ids, id)
		}
		tests, err := s.testRepo.GetByIDs(ids)
		if err != nil {
			return nil, err
		}
		if len(tests) != len(testIDSet) {
			return nil, newValidationError("one or more tests do not exist")
		}
	}

	for _, ref := range normalized {
		step := models.CourseTopicStep{
			StepType: ref.Type,
		}
		switch ref.Type {
		case models.CourseTopicStepTypeVideo:
			videoID := ref.ID
			step.VideoID = &videoID
		case models.CourseTopicStepTypeTest:
			testID := ref.ID
			step.TestID = &testID
		}
		steps = append(steps, step)
	}

	return steps, nil
}

func (s *TopicService) assignVideos(topicID uint, videoIDs []uint) error {
	if s == nil || s.topicRepo == nil {
		return errors.New("course topic repository is not configured")
	}
	refs := make([]models.CourseTopicStepReference, 0, len(videoIDs))
	for _, id := range videoIDs {
		refs = append(refs, models.CourseTopicStepReference{Type: models.CourseTopicStepTypeVideo, ID: id})
	}
	steps, err := s.buildSteps(refs)
	if err != nil {
		return err
	}
	return s.topicRepo.SetSteps(topicID, steps)
}

func (s *TopicService) populateSteps(topics []models.CourseTopic) error {
	if len(topics) == 0 {
		return nil
	}
	if s.topicRepo == nil {
		return errors.New("course topic repository is not configured")
	}

	topicIDs := make([]uint, 0, len(topics))
	for i := range topics {
		topics[i].Videos = []models.CourseVideo{}
		topics[i].Steps = []models.CourseTopicStep{}
		topicIDs = append(topicIDs, topics[i].ID)
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

	topicMap := make(map[uint]*models.CourseTopic, len(topics))
	for i := range topics {
		topicMap[topics[i].ID] = &topics[i]
	}

	for topicID, links := range linksByTopic {
		topic, ok := topicMap[topicID]
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
