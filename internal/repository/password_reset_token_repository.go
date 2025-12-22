package repository

import (
	"time"

	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
)

type PasswordResetTokenRepository interface {
	Create(token *models.PasswordResetToken) error
	GetActiveByHash(hash string, now time.Time) (*models.PasswordResetToken, error)
	MarkUsed(id uint, usedAt time.Time) error
	DeleteExpired(now time.Time) error
	DeleteByUser(userID uint) error
}

type passwordResetTokenRepository struct {
	db *gorm.DB
}

func NewPasswordResetTokenRepository(db *gorm.DB) PasswordResetTokenRepository {
	return &passwordResetTokenRepository{db: db}
}

func (r *passwordResetTokenRepository) Create(token *models.PasswordResetToken) error {
	return r.db.Create(token).Error
}

func (r *passwordResetTokenRepository) GetActiveByHash(hash string, now time.Time) (*models.PasswordResetToken, error) {
	var token models.PasswordResetToken
	err := r.db.Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, now).
		First(&token).Error
	return &token, err
}

func (r *passwordResetTokenRepository) MarkUsed(id uint, usedAt time.Time) error {
	return r.db.Model(&models.PasswordResetToken{}).
		Where("id = ?", id).
		Update("used_at", usedAt).Error
}

func (r *passwordResetTokenRepository) DeleteExpired(now time.Time) error {
	return r.db.Where("expires_at <= ? OR used_at IS NOT NULL", now).
		Delete(&models.PasswordResetToken{}).Error
}

func (r *passwordResetTokenRepository) DeleteByUser(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.PasswordResetToken{}).Error
}
