package service

import (
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"errors"
)

type CommentService struct {
	commentRepo repository.CommentRepository
}

func NewCommentService(commentRepo repository.CommentRepository) *CommentService {
	return &CommentService{commentRepo: commentRepo}
}

func (s *CommentService) Create(postID, authorID uint, req models.CreateCommentRequest) (*models.Comment, error) {
	comment := &models.Comment{
		Content:  req.Content,
		PostID:   postID,
		AuthorID: authorID,
		ParentID: req.ParentID,
		Approved: true,
	}

	if err := s.commentRepo.Create(comment); err != nil {
		return nil, err
	}

	return s.commentRepo.GetByID(comment.ID)
}

func (s *CommentService) GetByPostID(postID uint) ([]models.Comment, error) {
	return s.commentRepo.GetByPostID(postID)
}

func (s *CommentService) GetAll() ([]models.Comment, error) {
	return s.commentRepo.GetAll()
}

func (s *CommentService) Update(id, userID uint, isAdmin bool, req models.UpdateCommentRequest) (*models.Comment, error) {
	comment, err := s.commentRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !isAdmin && comment.AuthorID != userID {
		return nil, errors.New("unauthorized")
	}

	comment.Content = req.Content
	if req.Approved != nil && isAdmin {
		comment.Approved = *req.Approved
	}

	if err := s.commentRepo.Update(comment); err != nil {
		return nil, err
	}

	return s.commentRepo.GetByID(comment.ID)
}

func (s *CommentService) Delete(id, userID uint, isAdmin bool) error {
	comment, err := s.commentRepo.GetByID(id)
	if err != nil {
		return err
	}

	if !isAdmin && comment.AuthorID != userID {
		return errors.New("unauthorized")
	}

	return s.commentRepo.Delete(id)
}

func (s *CommentService) ApproveComment(commentID uint) error {
	comment, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		return err
	}

	comment.Approved = true
	return s.commentRepo.Update(comment)
}

func (s *CommentService) RejectComment(commentID uint) error {
	comment, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		return err
	}

	comment.Approved = false
	return s.commentRepo.Update(comment)
}
