package service

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
)

type TopicService struct {
	topicRepo repository.CourseTopicRepository
	videoRepo repository.CourseVideoRepository
}

func NewTopicService(topicRepo repository.CourseTopicRepository, videoRepo repository.CourseVideoRepository) *TopicService {
	return &TopicService{
		topicRepo: topicRepo,
		videoRepo: videoRepo,
	}
}

func (s *TopicService) SetRepositories(topicRepo repository.CourseTopicRepository, videoRepo repository.CourseVideoRepository) {
	if s == nil {
		return
	}
	s.topicRepo = topicRepo
	s.videoRepo = videoRepo
}

func (s *TopicService) Create(req models.CreateCourseTopicRequest) (*models.CourseTopic, error) {
	if s == nil || s.topicRepo == nil {
		return nil, errors.New("course topic repository is not configured")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, errors.New("topic title is required")
	}

	topic := models.CourseTopic{
		Title:       title,
		Description: strings.TrimSpace(req.Description),
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
		return nil, errors.New("topic title is required")
	}

	topic.Title = title
	topic.Description = strings.TrimSpace(req.Description)

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

	topics := []models.CourseTopic{*topic}
	if err := s.populateVideos(topics); err != nil {
		return nil, err
	}

	result := topics[0]
	return &result, nil
}

func (s *TopicService) List() ([]models.CourseTopic, error) {
	if s == nil || s.topicRepo == nil {
		return nil, errors.New("course topic repository is not configured")
	}

	topics, err := s.topicRepo.List()
	if err != nil {
		return nil, err
	}

	if err := s.populateVideos(topics); err != nil {
		return nil, err
	}

	return topics, nil
}

func (s *TopicService) UpdateVideos(topicID uint, videoIDs []uint) (*models.CourseTopic, error) {
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

	if err := s.assignVideos(topicID, videoIDs); err != nil {
		return nil, err
	}

	return s.GetByID(topicID)
}

func (s *TopicService) assignVideos(topicID uint, videoIDs []uint) error {
	if len(videoIDs) == 0 {
		return s.topicRepo.SetVideos(topicID, nil)
	}
	if s.videoRepo == nil {
		return errors.New("course video repository is not configured")
	}

	unique := uniqueOrdered(videoIDs)
	videos, err := s.videoRepo.GetByIDs(unique)
	if err != nil {
		return err
	}
	if len(videos) != len(unique) {
		return fmt.Errorf("one or more videos do not exist")
	}

	return s.topicRepo.SetVideos(topicID, unique)
}

func (s *TopicService) populateVideos(topics []models.CourseTopic) error {
	if len(topics) == 0 {
		return nil
	}
	if s.videoRepo == nil || s.topicRepo == nil {
		return errors.New("course video repository is not configured")
	}

	topicIDs := make([]uint, 0, len(topics))
	for i := range topics {
		topics[i].Videos = []models.CourseVideo{}
		topicIDs = append(topicIDs, topics[i].ID)
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

	ids := make([]uint, 0, len(videoIDSet))
	for id := range videoIDSet {
		ids = append(ids, id)
	}

	videos, err := s.videoRepo.GetByIDs(ids)
	if err != nil {
		return err
	}

	videoMap := make(map[uint]models.CourseVideo, len(videos))
	for _, video := range videos {
		videoMap[video.ID] = video
	}

	topicMap := make(map[uint]*models.CourseTopic, len(topics))
	for i := range topics {
		topic := &topics[i]
		topicMap[topic.ID] = topic
	}

	for topicID, links := range linksByTopic {
		topic, ok := topicMap[topicID]
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
