package service

import "errors"

var (
	ErrQuestionNotFound = errors.New("question not found")
	ErrAnswerNotFound   = errors.New("answer not found")
	ErrCategoryNotFound = errors.New("category not found")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrInvalidVoteValue = errors.New("invalid vote value")
)
