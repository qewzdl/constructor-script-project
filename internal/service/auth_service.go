package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"constructor-script-backend/internal/authorization"
	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/logger"
)

type AuthService struct {
	userRepo     repository.UserRepository
	resetRepo    repository.PasswordResetTokenRepository
	emailService *EmailService
	jwtSecret    string
	config       *config.Config
}

var (
	ErrIncorrectOldPassword  = errors.New("incorrect old password")
	ErrUserNotFound          = errors.New("user not found")
	ErrPasswordResetDisabled = errors.New("password reset is not available")
	ErrInvalidResetToken     = errors.New("invalid or expired reset token")
)

const passwordResetTTL = time.Hour

type validationError struct {
	message string
}

func (e validationError) Error() string {
	return e.message
}

func newValidationError(message string) error {
	return validationError{message: message}
}

func IsValidationError(err error) bool {
	var vErr validationError
	return errors.As(err, &vErr)
}

func NewAuthService(userRepo repository.UserRepository, resetRepo repository.PasswordResetTokenRepository, emailService *EmailService, jwtSecret string, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		resetRepo:    resetRepo,
		emailService: emailService,
		jwtSecret:    jwtSecret,
		config:       cfg,
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
		Role:     authorization.RoleUser,
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
		"role":     user.Role.String(),
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

func (s *AuthService) GetAllUsers(query string, limit int) ([]models.User, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed != "" {
		max := limit
		if max <= 0 || max > 100 {
			max = 25
		}
		return s.userRepo.Search(trimmed, max)
	}

	return s.userRepo.GetAll()
}

func (s *AuthService) DeleteUser(id uint) error {
	return s.userRepo.Delete(id)
}

func (s *AuthService) UpdateUserRole(id uint, role string) error {
	targetRole := authorization.UserRole(strings.ToLower(strings.TrimSpace(role)))
	if !targetRole.IsValid() {
		return errors.New("invalid role")
	}

	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return err
	}

	user.Role = targetRole
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return ErrIncorrectOldPassword
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

func (s *AuthService) RequestPasswordReset(email string) error {
	if s.resetRepo == nil || s.emailService == nil || !s.emailService.Enabled() {
		logger.Warn("Password reset disabled: email service not configured", map[string]interface{}{
			"enable_email":      s.config != nil && s.config.EnableEmail,
			"smtp_host_set":     s.config != nil && strings.TrimSpace(s.config.SMTPHost) != "",
			"smtp_username_set": s.config != nil && strings.TrimSpace(s.config.SMTPUsername) != "",
			"smtp_password_set": s.config != nil && strings.TrimSpace(s.config.SMTPPassword) != "",
		})
		return ErrPasswordResetDisabled
	}

	normalized := strings.TrimSpace(email)
	if normalized == "" {
		return newValidationError("email is required")
	}

	cfg := s.emailService.resolveConfig()
	logger.Info("Password reset email dispatch starting", map[string]interface{}{
		"email":             normalized,
		"enable_email":      s.config != nil && s.config.EnableEmail,
		"smtp_host":         cfg.Host,
		"smtp_port":         cfg.Port,
		"smtp_from":         cfg.From,
		"smtp_username_set": cfg.Username != "",
		"reset_repo_ready":  s.resetRepo != nil,
		"site_url": func() string {
			if s.config == nil {
				return ""
			}
			return strings.TrimSpace(s.config.SiteURL)
		}(),
	})

	user, err := s.userRepo.GetByEmail(normalized)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Info("Password reset requested for unknown email", map[string]interface{}{
				"email": normalized,
			})
			return nil
		}
		return err
	}

	if err := s.resetRepo.DeleteByUser(user.ID); err != nil {
		return fmt.Errorf("failed to prepare reset token: %w", err)
	}

	token, err := generateResetToken()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	expiresAt := time.Now().Add(passwordResetTTL)
	record := &models.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: hashResetToken(token),
		ExpiresAt: expiresAt,
	}

	if err := s.resetRepo.Create(record); err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	siteName := "your account"
	if s.config != nil && strings.TrimSpace(s.config.SiteName) != "" {
		siteName = strings.TrimSpace(s.config.SiteName)
	}

	resetURL := s.buildResetURL(token)
	subject := fmt.Sprintf("Reset your %s password", siteName)
	body := fmt.Sprintf(
		"We received a request to reset your password for %s.\n\nUse the link below to set a new password. The link will expire in %d minutes.\n\n%s\n\nIf you did not request this, you can ignore this email.",
		siteName, int(passwordResetTTL.Minutes()), resetURL,
	)

	if err := s.emailService.Send(user.Email, subject, body); err != nil {
		logger.Error(err, "Failed to send password reset email", map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
		})
		return fmt.Errorf("failed to send reset email: %w", err)
	}

	go s.cleanupExpiredTokens()

	return nil
}

func (s *AuthService) ResetPassword(token, newPassword string) error {
	if s.resetRepo == nil {
		return ErrPasswordResetDisabled
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return newValidationError("reset token is required")
	}

	if err := validatePasswordStrength(newPassword); err != nil {
		return err
	}

	now := time.Now()
	record, err := s.resetRepo.GetActiveByHash(hashResetToken(token), now)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidResetToken
		}
		return err
	}

	user, err := s.userRepo.GetByID(record.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidResetToken
		}
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	if err := s.userRepo.Update(user); err != nil {
		return err
	}

	if err := s.resetRepo.MarkUsed(record.ID, now); err != nil {
		return err
	}

	_ = s.resetRepo.DeleteExpired(now)

	return nil
}

func (s *AuthService) buildResetURL(token string) string {
	baseURL := ""
	if s.config != nil {
		baseURL = strings.TrimRight(strings.TrimSpace(s.config.SiteURL), "/")
	}

	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}

	return fmt.Sprintf("%s/reset-password?token=%s", baseURL, token)
}

func (s *AuthService) cleanupExpiredTokens() {
	if s.resetRepo == nil {
		return
	}
	_ = s.resetRepo.DeleteExpired(time.Now())
}

func generateResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func validatePasswordStrength(password string) error {
	if len(password) < 6 {
		return newValidationError("password must be at least 6 characters long")
	}

	return nil
}
