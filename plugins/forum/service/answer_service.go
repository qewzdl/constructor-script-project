package service

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
)

type AnswerService struct {
	answerRepo   repository.ForumAnswerRepository
	questionRepo repository.ForumQuestionRepository
	voteRepo     repository.ForumAnswerVoteRepository
}

func NewAnswerService(answerRepo repository.ForumAnswerRepository, questionRepo repository.ForumQuestionRepository, voteRepo repository.ForumAnswerVoteRepository) *AnswerService {
	svc := &AnswerService{}
	svc.SetRepositories(answerRepo, questionRepo, voteRepo)
	return svc
}

func (s *AnswerService) SetRepositories(answerRepo repository.ForumAnswerRepository, questionRepo repository.ForumQuestionRepository, voteRepo repository.ForumAnswerVoteRepository) {
	if s == nil {
		return
	}
	s.answerRepo = answerRepo
	s.questionRepo = questionRepo
	s.voteRepo = voteRepo
}

func (s *AnswerService) Create(questionID, authorID uint, req models.CreateForumAnswerRequest) (*models.ForumAnswer, error) {
	if s == nil || s.answerRepo == nil || s.questionRepo == nil {
		return nil, errors.New("answer service not configured")
	}
	if _, err := s.questionRepo.GetByID(questionID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}

	cleanedContent := strings.TrimSpace(req.Content)
	if cleanedContent == "" {
		return nil, errors.New("answer content is required")
	}

	answer := &models.ForumAnswer{
		QuestionID: questionID,
		AuthorID:   authorID,
		Content:    cleanedContent,
	}

	if err := s.answerRepo.Create(answer); err != nil {
		return nil, fmt.Errorf("failed to create answer: %w", err)
	}
	return s.answerRepo.GetByID(answer.ID)
}

func (s *AnswerService) Update(id uint, req models.UpdateForumAnswerRequest, userID uint, canManageAll bool) (*models.ForumAnswer, error) {
	if s == nil || s.answerRepo == nil {
		return nil, errors.New("answer repository not configured")
	}

	answer, err := s.answerRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnswerNotFound
		}
		return nil, err
	}

	if !canManageAll && answer.AuthorID != userID {
		return nil, ErrUnauthorized
	}

	if req.Content != nil {
		cleaned := strings.TrimSpace(*req.Content)
		if cleaned == "" {
			return nil, errors.New("answer content cannot be empty")
		}
		answer.Content = cleaned
	}

	if err := s.answerRepo.Update(answer); err != nil {
		return nil, fmt.Errorf("failed to update answer: %w", err)
	}

	return s.answerRepo.GetByID(answer.ID)
}

func (s *AnswerService) Delete(id uint, userID uint, canManageAll bool) error {
	if s == nil || s.answerRepo == nil {
		return errors.New("answer repository not configured")
	}
	answer, err := s.answerRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAnswerNotFound
		}
		return err
	}
	if !canManageAll && answer.AuthorID != userID {
		return ErrUnauthorized
	}
	return s.answerRepo.Delete(id)
}

func (s *AnswerService) Vote(answerID, userID uint, value int) (int, error) {
	if s == nil || s.answerRepo == nil || s.voteRepo == nil {
		return 0, errors.New("answer voting not configured")
	}
	if value < -1 || value > 1 {
		return 0, ErrInvalidVoteValue
	}
	if _, err := s.answerRepo.GetByID(answerID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrAnswerNotFound
		}
		return 0, err
	}
	if value == 0 {
		return s.voteRepo.RemoveVote(answerID, userID)
	}
	return s.voteRepo.SetVote(answerID, userID, value)
}

func (s *AnswerService) ListByQuestion(questionID uint) ([]models.ForumAnswer, error) {
	if s == nil || s.answerRepo == nil {
		return nil, errors.New("answer repository not configured")
	}
	return s.answerRepo.ListByQuestion(questionID)
}

func (s *AnswerService) Get(id uint) (*models.ForumAnswer, error) {
	if s == nil || s.answerRepo == nil {
		return nil, errors.New("answer repository not configured")
	}
	answer, err := s.answerRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnswerNotFound
		}
		return nil, err
	}
	return answer, nil
}
