package service

import (
	"errors"
	"math"
	"mime/multipart"
	"os"
	"strings"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/internal/service"
)

type VideoService struct {
	videoRepo     repository.CourseVideoRepository
	uploadService *service.UploadService
}

func NewVideoService(videoRepo repository.CourseVideoRepository, uploadService *service.UploadService) *VideoService {
	return &VideoService{
		videoRepo:     videoRepo,
		uploadService: uploadService,
	}
}

func (s *VideoService) SetUploadService(uploadService *service.UploadService) {
	if s == nil {
		return
	}
	s.uploadService = uploadService
}

func (s *VideoService) Create(req models.CreateCourseVideoRequest, file *multipart.FileHeader) (*models.CourseVideo, error) {
	if s == nil || s.videoRepo == nil {
		return nil, errors.New("course video repository is not configured")
	}
	if s.uploadService == nil {
		return nil, errors.New("upload service is not configured")
	}
	if file == nil {
		return nil, newValidationError("video file is required")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, newValidationError("video title is required")
	}

	url, filename, duration, err := s.uploadService.UploadVideo(file, req.Preferred)
	if err != nil {
		return nil, err
	}

	seconds := int(math.Round(duration.Seconds()))
	if seconds <= 0 && duration > 0 {
		seconds = 1
	}

	video := models.CourseVideo{
		Title:           title,
		Description:     strings.TrimSpace(req.Description),
		FileURL:         url,
		Filename:        filename,
		DurationSeconds: seconds,
	}

	if err := s.videoRepo.Create(&video); err != nil {
		s.uploadService.DeleteImage(url)
		return nil, err
	}

	return &video, nil
}

func (s *VideoService) Update(id uint, req models.UpdateCourseVideoRequest) (*models.CourseVideo, error) {
	if s == nil || s.videoRepo == nil {
		return nil, errors.New("course video repository is not configured")
	}

	video, err := s.videoRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, newValidationError("video title is required")
	}

	video.Title = title
	video.Description = strings.TrimSpace(req.Description)

	if err := s.videoRepo.Update(video); err != nil {
		return nil, err
	}

	return video, nil
}

func (s *VideoService) Delete(id uint) error {
	if s == nil || s.videoRepo == nil {
		return errors.New("course video repository is not configured")
	}

	video, err := s.videoRepo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.videoRepo.Delete(id); err != nil {
		return err
	}

	if s.uploadService != nil {
		if err := s.uploadService.DeleteImage(video.FileURL); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	return nil
}

func (s *VideoService) GetByID(id uint) (*models.CourseVideo, error) {
	if s == nil || s.videoRepo == nil {
		return nil, errors.New("course video repository is not configured")
	}
	return s.videoRepo.GetByID(id)
}

func (s *VideoService) List() ([]models.CourseVideo, error) {
	if s == nil || s.videoRepo == nil {
		return nil, errors.New("course video repository is not configured")
	}
	return s.videoRepo.List()
}

func (s *VideoService) Exists(id uint) (bool, error) {
	if s == nil || s.videoRepo == nil {
		return false, errors.New("course video repository is not configured")
	}
	return s.videoRepo.Exists(id)
}
