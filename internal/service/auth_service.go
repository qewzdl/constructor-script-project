package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"time"
	"unicode/utf8"

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
	userRepo      repository.UserRepository
	resetRepo     repository.PasswordResetTokenRepository
	emailService  *EmailService
	uploadService *UploadService
	jwtSecret     string
	config        *config.Config
	settingRepo   repository.SettingRepository
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

func NewAuthService(
	userRepo repository.UserRepository,
	resetRepo repository.PasswordResetTokenRepository,
	emailService *EmailService,
	settingRepo repository.SettingRepository,
	uploadService *UploadService,
	jwtSecret string,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		resetRepo:     resetRepo,
		emailService:  emailService,
		uploadService: uploadService,
		jwtSecret:     jwtSecret,
		config:        cfg,
		settingRepo:   settingRepo,
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

	if err := s.ensureUserAvatar(user); err != nil {
		logger.Warn("Failed to assign placeholder avatar for new user", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
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

	if err := s.ensureUserAvatar(user); err != nil {
		logger.Warn("Failed to assign placeholder avatar during login", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
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
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if err := s.ensureUserAvatar(user); err != nil {
		logger.Warn("Failed to ensure user avatar", map[string]interface{}{
			"user_id": id,
			"error":   err.Error(),
		})
	}

	return user, nil
}

func (s *AuthService) UpdateProfile(userID uint, username, email string, avatar *string) (*models.User, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	currentAvatar := strings.TrimSpace(user.Avatar)
	avatarChanged := false
	var oldAvatar string

	trimmedUsername := strings.TrimSpace(username)
	if trimmedUsername != "" && trimmedUsername != user.Username {
		existingUser, _ := s.userRepo.GetByUsername(trimmedUsername)
		if existingUser != nil && existingUser.ID != user.ID {
			return nil, errors.New("username already taken")
		}
		user.Username = trimmedUsername
	}

	trimmedEmail := strings.TrimSpace(email)
	if trimmedEmail != "" && trimmedEmail != user.Email {
		existingUser, _ := s.userRepo.GetByEmail(trimmedEmail)
		if existingUser != nil && existingUser.ID != user.ID {
			return nil, errors.New("email already taken")
		}
		user.Email = trimmedEmail
	}

	if avatar != nil {
		newAvatar := strings.TrimSpace(*avatar)
		if newAvatar != currentAvatar {
			oldAvatar = currentAvatar
			user.Avatar = newAvatar
			avatarChanged = true
		}
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	if avatarChanged && s.uploadService != nil && oldAvatar != "" && oldAvatar != user.Avatar && s.uploadService.IsManagedURL(oldAvatar) && !s.uploadService.IsInitialAvatar(oldAvatar) {
		if err := s.uploadService.DeleteUpload(oldAvatar); err != nil {
			logger.Warn("Failed to delete old avatar", map[string]interface{}{"user_id": user.ID, "avatar": oldAvatar, "error": err.Error()})
		}
	}

	if err := s.ensureUserAvatar(user); err != nil {
		logger.Warn("Failed to ensure user avatar after update", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
	}

	return user, nil
}

func (s *AuthService) UploadAvatar(userID uint, file *multipart.FileHeader) (*models.User, error) {
	if s.uploadService == nil {
		return nil, errUploadServiceMissing
	}
	if file == nil {
		return nil, ErrUploadMissing
	}

	preferredName := fmt.Sprintf("avatar-%d", userID)
	url, _, err := s.uploadService.UploadImage(file, preferredName)
	if err != nil {
		return nil, err
	}

	user, updateErr := s.UpdateProfile(userID, "", "", &url)
	if updateErr != nil {
		if s.uploadService.IsManagedURL(url) {
			if deleteErr := s.uploadService.DeleteUpload(url); deleteErr != nil {
				logger.Warn("Failed to rollback uploaded avatar after update error", map[string]interface{}{
					"user_id": userID,
					"avatar":  url,
					"error":   deleteErr.Error(),
				})
			}
		}
		return nil, updateErr
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
	user, err := s.GetUserByID(userID)
	if err != nil {
		return "", nil, err
	}

	newToken, err := s.generateToken(user)
	if err != nil {
		return "", nil, err
	}

	return newToken, user, nil
}

func (s *AuthService) ensureUserAvatar(user *models.User) error {
	if s == nil || user == nil {
		return nil
	}

	if s.uploadService == nil {
		return nil
	}

	if strings.TrimSpace(user.Avatar) != "" {
		return nil
	}

	initial := ""
	if trimmed := strings.TrimSpace(user.Username); trimmed != "" {
		r, _ := utf8.DecodeRuneInString(trimmed)
		if r != utf8.RuneError {
			initial = string(r)
		}
	}

	if initial == "" {
		return nil
	}

	url, err := s.uploadService.EnsureInitialAvatar(initial)
	if err != nil {
		return err
	}

	if strings.TrimSpace(url) == "" || strings.TrimSpace(user.Avatar) == url {
		return nil
	}

	user.Avatar = url
	return s.userRepo.Update(user)
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
	siteName, baseURL := s.resolveSiteMeta()
	logger.Info("Password reset email dispatch starting", map[string]interface{}{
		"email":             normalized,
		"enable_email":      s.config != nil && s.config.EnableEmail,
		"smtp_host":         cfg.Host,
		"smtp_port":         cfg.Port,
		"smtp_from":         cfg.From,
		"smtp_username_set": cfg.Username != "",
		"reset_repo_ready":  s.resetRepo != nil,
		"site_url_resolved": baseURL,
		"site_name":         siteName,
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

	resetURL := s.buildResetURL(baseURL, token)
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

func (s *AuthService) buildResetURL(baseURL, token string) string {
	return fmt.Sprintf("%s/reset-password?token=%s", baseURL, token)
}

func (s *AuthService) resolveSiteMeta() (siteName, baseURL string) {
	siteName = "your account"
	baseURL = ""

	if s.config != nil {
		if trimmedName := strings.TrimSpace(s.config.SiteName); trimmedName != "" {
			siteName = trimmedName
		}
		baseURL = strings.TrimRight(strings.TrimSpace(s.config.SiteURL), "/")
	}

	if s.settingRepo != nil {
		if setting, err := s.settingRepo.Get(settingKeySiteName); err == nil {
			if value := strings.TrimSpace(setting.Value); value != "" {
				siteName = value
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Failed to read site name from settings", map[string]interface{}{
				"error": err.Error(),
			})
		}

		if setting, err := s.settingRepo.Get(settingKeySiteURL); err == nil {
			if value := strings.TrimRight(strings.TrimSpace(setting.Value), "/"); value != "" {
				baseURL = value
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Failed to read site URL from settings", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	if strings.TrimSpace(siteName) == "" {
		siteName = "your account"
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "http://localhost:8081"
	}

	return siteName, baseURL
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
