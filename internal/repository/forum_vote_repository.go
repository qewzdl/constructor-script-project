package repository

import (
	"errors"

	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
)

type ForumQuestionVoteRepository interface {
	SetVote(questionID, userID uint, value int) (int, error)
	RemoveVote(questionID, userID uint) (int, error)
}

type ForumAnswerVoteRepository interface {
	SetVote(answerID, userID uint, value int) (int, error)
	RemoveVote(answerID, userID uint) (int, error)
}

type forumQuestionVoteRepository struct {
	db *gorm.DB
}

type forumAnswerVoteRepository struct {
	db *gorm.DB
}

func NewForumQuestionVoteRepository(db *gorm.DB) ForumQuestionVoteRepository {
	return &forumQuestionVoteRepository{db: db}
}

func NewForumAnswerVoteRepository(db *gorm.DB) ForumAnswerVoteRepository {
	return &forumAnswerVoteRepository{db: db}
}

func (r *forumQuestionVoteRepository) SetVote(questionID, userID uint, value int) (int, error) {
	if r == nil || r.db == nil {
		return 0, gorm.ErrInvalidDB
	}
	var rating int
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var question models.ForumQuestion
		if err := tx.Select("id").First(&question, questionID).Error; err != nil {
			return err
		}

		var vote models.ForumQuestionVote
		result := tx.Where("question_id = ? AND user_id = ?", questionID, userID).First(&vote)
		delta := 0
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			vote = models.ForumQuestionVote{QuestionID: questionID, UserID: userID, Value: value}
			if err := tx.Create(&vote).Error; err != nil {
				return err
			}
			delta = value
		} else if result.Error != nil {
			return result.Error
		} else {
			if vote.Value == value {
				delta = 0
			} else {
				delta = value - vote.Value
				vote.Value = value
				if err := tx.Save(&vote).Error; err != nil {
					return err
				}
			}
		}

		if delta != 0 {
			if err := tx.Model(&models.ForumQuestion{}).Where("id = ?", questionID).UpdateColumn("rating", gorm.Expr("rating + ?", delta)).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&models.ForumQuestion{}).Where("id = ?", questionID).Select("rating").First(&question).Error; err != nil {
			return err
		}
		rating = question.Rating
		return nil
	})
	if err != nil {
		return 0, err
	}
	return rating, nil
}

func (r *forumQuestionVoteRepository) RemoveVote(questionID, userID uint) (int, error) {
	if r == nil || r.db == nil {
		return 0, gorm.ErrInvalidDB
	}
	var rating int
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var question models.ForumQuestion
		if err := tx.Select("id").First(&question, questionID).Error; err != nil {
			return err
		}

		var vote models.ForumQuestionVote
		if err := tx.Where("question_id = ? AND user_id = ?", questionID, userID).First(&vote).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Model(&models.ForumQuestion{}).Where("id = ?", questionID).Select("rating").First(&question).Error; err != nil {
					return err
				}
				rating = question.Rating
				return nil
			}
			return err
		}

		if err := tx.Delete(&vote).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.ForumQuestion{}).Where("id = ?", questionID).UpdateColumn("rating", gorm.Expr("rating + ?", -vote.Value)).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.ForumQuestion{}).Where("id = ?", questionID).Select("rating").First(&question).Error; err != nil {
			return err
		}
		rating = question.Rating
		return nil
	})
	if err != nil {
		return 0, err
	}
	return rating, nil
}

func (r *forumAnswerVoteRepository) SetVote(answerID, userID uint, value int) (int, error) {
	if r == nil || r.db == nil {
		return 0, gorm.ErrInvalidDB
	}
	var rating int
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var answer models.ForumAnswer
		if err := tx.Select("id").First(&answer, answerID).Error; err != nil {
			return err
		}

		var vote models.ForumAnswerVote
		result := tx.Where("answer_id = ? AND user_id = ?", answerID, userID).First(&vote)
		delta := 0
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			vote = models.ForumAnswerVote{AnswerID: answerID, UserID: userID, Value: value}
			if err := tx.Create(&vote).Error; err != nil {
				return err
			}
			delta = value
		} else if result.Error != nil {
			return result.Error
		} else {
			if vote.Value == value {
				delta = 0
			} else {
				delta = value - vote.Value
				vote.Value = value
				if err := tx.Save(&vote).Error; err != nil {
					return err
				}
			}
		}

		if delta != 0 {
			if err := tx.Model(&models.ForumAnswer{}).Where("id = ?", answerID).UpdateColumn("rating", gorm.Expr("rating + ?", delta)).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&models.ForumAnswer{}).Where("id = ?", answerID).Select("rating").First(&answer).Error; err != nil {
			return err
		}
		rating = answer.Rating
		return nil
	})
	if err != nil {
		return 0, err
	}
	return rating, nil
}

func (r *forumAnswerVoteRepository) RemoveVote(answerID, userID uint) (int, error) {
	if r == nil || r.db == nil {
		return 0, gorm.ErrInvalidDB
	}
	var rating int
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var answer models.ForumAnswer
		if err := tx.Select("id").First(&answer, answerID).Error; err != nil {
			return err
		}

		var vote models.ForumAnswerVote
		if err := tx.Where("answer_id = ? AND user_id = ?", answerID, userID).First(&vote).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Model(&models.ForumAnswer{}).Where("id = ?", answerID).Select("rating").First(&answer).Error; err != nil {
					return err
				}
				rating = answer.Rating
				return nil
			}
			return err
		}

		if err := tx.Delete(&vote).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.ForumAnswer{}).Where("id = ?", answerID).UpdateColumn("rating", gorm.Expr("rating + ?", -vote.Value)).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.ForumAnswer{}).Where("id = ?", answerID).Select("rating").First(&answer).Error; err != nil {
			return err
		}
		rating = answer.Rating
		return nil
	})
	if err != nil {
		return 0, err
	}
	return rating, nil
}
