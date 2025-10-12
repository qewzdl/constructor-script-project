package repository

import (
	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.User) error
	GetByID(id uint) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	GetAll() ([]models.User, error)
	Update(user *models.User) error
	Delete(id uint) error
	Count() (int64, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	return &user, err
}

func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	return &user, err
}

func (r *userRepository) GetAll() ([]models.User, error) {
	var users []models.User
	err := r.db.Find(&users).Error
	return users, err
}

func (r *userRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) Delete(id uint) error {
	return r.db.Unscoped().Delete(&models.User{}, id).Error
}

func (r *userRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Count(&count).Error
	return count, err
}

func (r *userRepository) GetWithStats(id uint) (*models.User, error) {
	type UserWithStats struct {
		models.User
		PostCount    int `json:"post_count"`
		CommentCount int `json:"comment_count"`
	}

	var userStats UserWithStats
	err := r.db.Model(&models.User{}).
		Select("users.*, COUNT(DISTINCT posts.id) as post_count, COUNT(DISTINCT comments.id) as comment_count").
		Joins("LEFT JOIN posts ON posts.author_id = users.id").
		Joins("LEFT JOIN comments ON comments.author_id = users.id").
		Where("users.id = ?", id).
		Group("users.id").
		First(&userStats).Error

	return &userStats.User, err
}

func (r *userRepository) Search(query string, limit int) ([]models.User, error) {
	var users []models.User
	err := r.db.Where("username ILIKE ? OR email ILIKE ?", "%"+query+"%", "%"+query+"%").
		Limit(limit).
		Find(&users).Error
	return users, err
}
