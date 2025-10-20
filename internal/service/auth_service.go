package service

import (
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
)

type AuthService struct {
	userRepo  repository.UserRepository
	jwtSecret string
}

func NewAuthService(userRepo repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (s *AuthService) Register(req models.RegisterRequest) (*models.User, error) {
	existingUser, err := s.userRepo.GetByEmail(req.Email)
	if err == nil && existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	existingUser, err = s.userRepo.GetByUsername(req.Username)
	if err == nil && existingUser != nil {
		return nil, errors.New("user with this username already exists")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := validatePasswordStrength(req.Password); err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		Role:     "user",
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(req models.LoginRequest) (string, *models.User, error) {
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

func (s *AuthService) generateToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"email":    user.Email,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(s.jwtSecret), nil
	})
}

func (s *AuthService) GetAllUsers() ([]models.User, error) {
	return s.userRepo.GetAll()
}

func (s *AuthService) DeleteUser(id uint) error {
	return s.userRepo.Delete(id)
}

func (s *AuthService) UpdateUserRole(id uint, role string) error {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return err
	}

	user.Role = role
	return s.userRepo.Update(user)
}

func (s *AuthService) GetUserByID(id uint) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

func (s *AuthService) UpdateProfile(userID uint, username, email string) (*models.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	if username != user.Username {
		existingUser, _ := s.userRepo.GetByUsername(username)
		if existingUser != nil {
			return nil, errors.New("username already taken")
		}
		user.Username = username
	}

	if email != user.Email {
		existingUser, _ := s.userRepo.GetByEmail(email)
		if existingUser != nil {
			return nil, errors.New("email already taken")
		}
		user.Email = email
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("incorrect old password")
	}

	if err := validatePasswordStrength(newPassword); err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	return s.userRepo.Update(user)
}

func (s *AuthService) RefreshToken(refreshToken string) (string, *models.User, error) {
	token, err := s.ValidateToken(refreshToken)
	if err != nil {
		return "", nil, errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", nil, errors.New("invalid token claims")
	}

	userID := uint(claims["user_id"].(float64))
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return "", nil, err
	}

	newToken, err := s.generateToken(user)
	if err != nil {
		return "", nil, err
	}

	return newToken, user, nil
}

func (s *AuthService) UpdateUserStatus(userID uint, status string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	user.Status = status
	return s.userRepo.Update(user)
}

func validatePasswordStrength(password string) error {
	var requirements []string

	if len(password) < 12 {
		requirements = append(requirements, "be at least 12 characters long")
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		requirements = append(requirements, "contain at least one uppercase letter")
	}
	if !hasLower {
		requirements = append(requirements, "contain at least one lowercase letter")
	}
	if !hasNumber {
		requirements = append(requirements, "include at least one digit")
	}
	if !hasSpecial {
		requirements = append(requirements, "include at least one special character")
	}

	if len(requirements) > 0 {
		return errors.New("password must " + strings.Join(requirements, ", "))
	}

	return nil
}
