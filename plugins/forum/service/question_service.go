package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/utils"
)

type QuestionService struct {
	questionRepo repository.ForumQuestionRepository
	categoryRepo repository.ForumCategoryRepository
	voteRepo     repository.ForumQuestionVoteRepository
}

func NewQuestionService(
	questionRepo repository.ForumQuestionRepository,
	categoryRepo repository.ForumCategoryRepository,
	voteRepo repository.ForumQuestionVoteRepository,
) *QuestionService {
	svc := &QuestionService{}
	svc.SetRepositories(questionRepo, categoryRepo, voteRepo)
	return svc
}

func (s *QuestionService) SetRepositories(
	questionRepo repository.ForumQuestionRepository,
	categoryRepo repository.ForumCategoryRepository,
	voteRepo repository.ForumQuestionVoteRepository,
) {
	if s == nil {
		return
	}
	s.questionRepo = questionRepo
	s.categoryRepo = categoryRepo
	s.voteRepo = voteRepo
}

type QuestionListOptions struct {
	Search       string
	AuthorID     *uint
	CategoryID   *uint
	CategorySlug string
}

func (s *QuestionService) List(page, limit int, opts QuestionListOptions) ([]models.ForumQuestion, int64, error) {
	if s == nil || s.questionRepo == nil {
		return nil, 0, errors.New("question repository not configured")
	}
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	var categoryID *uint
	if opts.CategoryID != nil {
		categoryID = opts.CategoryID
	} else if slug := strings.TrimSpace(opts.CategorySlug); slug != "" {
		if s.categoryRepo == nil {
			return nil, 0, errors.New("category repository not configured")
		}
		category, err := s.categoryRepo.GetBySlug(slug)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return []models.ForumQuestion{}, 0, nil
			}
			return nil, 0, err
		}
		id := category.ID
		categoryID = &id
	}

	search := strings.TrimSpace(opts.Search)
	return s.questionRepo.List(offset, limit, search, opts.AuthorID, categoryID)
}

func (s *QuestionService) GetByID(id uint) (*models.ForumQuestion, error) {
	if s == nil || s.questionRepo == nil {
		return nil, errors.New("question repository not configured")
	}
	question, err := s.questionRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}
	if err := s.questionRepo.IncrementViews(id); err != nil {
		return nil, fmt.Errorf("failed to update question views: %w", err)
	}
	question.Views++
	return question, nil
}

func (s *QuestionService) GetByIDWithoutIncrement(id uint) (*models.ForumQuestion, error) {
	if s == nil || s.questionRepo == nil {
		return nil, errors.New("question repository not configured")
	}
	question, err := s.questionRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}
	return question, nil
}

func (s *QuestionService) GetBySlug(slug string) (*models.ForumQuestion, error) {
	if s == nil || s.questionRepo == nil {
		return nil, errors.New("question repository not configured")
	}
	question, err := s.questionRepo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}
	if err := s.questionRepo.IncrementViews(question.ID); err != nil {
		return nil, fmt.Errorf("failed to update question views: %w", err)
	}
	question.Views++
	return question, nil
}

func (s *QuestionService) Create(req models.CreateForumQuestionRequest, authorID uint) (*models.ForumQuestion, error) {
	if s == nil || s.questionRepo == nil {
		return nil, errors.New("question repository not configured")
	}
	cleanedTitle := strings.TrimSpace(req.Title)
	if cleanedTitle == "" {
		return nil, errors.New("question title is required")
	}
	cleanedContent := strings.TrimSpace(req.Content)
	if cleanedContent == "" {
		return nil, errors.New("question content is required")
	}

	slug, err := s.generateUniqueSlug(cleanedTitle)
	if err != nil {
		return nil, err
	}

	categoryID, err := s.resolveCategoryID(req.CategoryID)
	if err != nil {
		return nil, err
	}

	question := &models.ForumQuestion{
		Title:      cleanedTitle,
		Slug:       slug,
		Content:    cleanedContent,
		AuthorID:   authorID,
		CategoryID: categoryID,
	}

	if err := s.questionRepo.Create(question); err != nil {
		return nil, fmt.Errorf("failed to create question: %w", err)
	}
	return s.questionRepo.GetByID(question.ID)
}

func (s *QuestionService) Update(id uint, req models.UpdateForumQuestionRequest, userID uint, canManageAll bool) (*models.ForumQuestion, error) {
	if s == nil || s.questionRepo == nil {
		return nil, errors.New("question repository not configured")
	}
	question, err := s.questionRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}

	if !canManageAll && question.AuthorID != userID {
		return nil, ErrUnauthorized
	}

	if req.Title != nil {
		cleaned := strings.TrimSpace(*req.Title)
		if cleaned == "" {
			return nil, errors.New("question title cannot be empty")
		}
		if cleaned != question.Title {
			slug, slugErr := s.generateUniqueSlug(cleaned)
			if slugErr != nil {
				return nil, slugErr
			}
			question.Title = cleaned
			question.Slug = slug
		}
	}

	if req.Content != nil {
		cleaned := strings.TrimSpace(*req.Content)
		if cleaned == "" {
			return nil, errors.New("question content cannot be empty")
		}
		question.Content = cleaned
	}

	if req.CategoryID.Set {
		categoryID, categoryErr := s.resolveCategoryID(req.CategoryID.Pointer())
		if categoryErr != nil {
			return nil, categoryErr
		}
		question.CategoryID = categoryID
	}

	if err := s.questionRepo.Update(question); err != nil {
		return nil, fmt.Errorf("failed to update question: %w", err)
	}

	return s.questionRepo.GetByID(question.ID)
}

func (s *QuestionService) Delete(id uint, userID uint, canManageAll bool) error {
	if s == nil || s.questionRepo == nil {
		return errors.New("question repository not configured")
	}
	question, err := s.questionRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrQuestionNotFound
		}
		return err
	}
	if !canManageAll && question.AuthorID != userID {
		return ErrUnauthorized
	}
	return s.questionRepo.Delete(id)
}

func (s *QuestionService) Vote(questionID, userID uint, value int) (int, error) {
	if s == nil || s.questionRepo == nil || s.voteRepo == nil {
		return 0, errors.New("question voting not configured")
	}
	if value < -1 || value > 1 {
		return 0, ErrInvalidVoteValue
	}
	if _, err := s.questionRepo.GetByID(questionID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrQuestionNotFound
		}
		return 0, err
	}
	if value == 0 {
		return s.voteRepo.RemoveVote(questionID, userID)
	}
	return s.voteRepo.SetVote(questionID, userID, value)
}

func (s *QuestionService) resolveCategoryID(raw *uint) (*uint, error) {
	if raw == nil {
		return nil, nil
	}
	id := *raw
	if id == 0 {
		return nil, nil
	}
	if s.categoryRepo == nil {
		return nil, errors.New("category repository not configured")
	}
	category, err := s.categoryRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to verify category: %w", err)
	}
	if category == nil {
		return nil, ErrCategoryNotFound
	}
	value := category.ID
	return &value, nil
}

func (s *QuestionService) generateUniqueSlug(title string) (string, error) {
	if s == nil || s.questionRepo == nil {
		return "", errors.New("question repository not configured")
	}
	base := utils.GenerateSlug(title)
	if base == "" {
		base = fmt.Sprintf("question-%d", time.Now().Unix())
	}
	slug := base
	for attempt := 1; attempt < 1000; attempt++ {
		exists, err := s.questionRepo.ExistsBySlug(slug)
		if err != nil {
			return "", fmt.Errorf("failed to validate slug availability: %w", err)
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, attempt)
	}
	return "", errors.New("failed to generate unique slug for question")
}
