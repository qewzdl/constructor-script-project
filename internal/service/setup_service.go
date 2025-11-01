package service

import (
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"constructor-script-backend/internal/authorization"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/lang"
	blogservice "constructor-script-backend/plugins/blog/service"
)

var (
	// ErrSetupAlreadyCompleted is returned when setup has already been completed.
	ErrSetupAlreadyCompleted = errors.New("setup already completed")
)

type InvalidFaviconError struct {
	Reason string
}

func (e *InvalidFaviconError) Error() string {
	if e == nil {
		return ""
	}

	return e.Reason
}

type InvalidLogoError struct {
	Reason string
}

func (e *InvalidLogoError) Error() string {
	if e == nil {
		return ""
	}

	return e.Reason
}

type SetupService struct {
	userRepo      repository.UserRepository
	settingRepo   repository.SettingRepository
	uploadService *UploadService
}

func NewSetupService(userRepo repository.UserRepository, settingRepo repository.SettingRepository, uploadService *UploadService) *SetupService {
	return &SetupService{
		userRepo:      userRepo,
		settingRepo:   settingRepo,
		uploadService: uploadService,
	}
}

func (s *SetupService) IsSetupComplete() (bool, error) {
	if s.userRepo == nil {
		return true, nil
	}

	count, err := s.userRepo.Count()
	if err != nil {
		return false, err
	}

	if count == 0 {
		return false, nil
	}

	if s.settingRepo == nil {
		return true, nil
	}

	setting, err := s.settingRepo.Get(settingKeySetupComplete)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, err
	}

	return setting.Value == "true", nil
}

func (s *SetupService) CompleteSetup(req models.SetupRequest, defaults models.SiteSettings) (*models.User, error) {
	if s.userRepo == nil {
		return nil, errors.New("user repository not configured")
	}

	count, err := s.userRepo.Count()
	if err != nil {
		return nil, err
	}

	if count > 0 {
		return nil, ErrSetupAlreadyCompleted
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username: req.AdminUsername,
		Email:    req.AdminEmail,
		Password: string(hashedPassword),
		Role:     authorization.RoleAdmin,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	if s.settingRepo != nil {
		if err := s.saveSiteSettings(req, defaults); err != nil {
			return nil, err
		}

		if err := s.settingRepo.Set(settingKeySetupComplete, "true"); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *SetupService) saveSiteSettings(req models.SetupRequest, defaults models.SiteSettings) error {
	settings := map[string]string{
		settingKeySiteName:          req.SiteName,
		settingKeySiteDescription:   req.SiteDescription,
		settingKeySiteURL:           req.SiteURL,
		settingKeySiteFavicon:       req.SiteFavicon,
		settingKeySiteLogo:          req.SiteLogo,
		settingKeyTagRetentionHours: strconv.Itoa(blogservice.DefaultUnusedTagRetentionHours),
	}

	for key, value := range settings {
		if strings.TrimSpace(value) == "" {
			continue
		}

		if err := s.settingRepo.Set(key, strings.TrimSpace(value)); err != nil {
			return err
		}
	}

	defaultLanguage := strings.TrimSpace(req.SiteDefaultLanguage)
	if defaultLanguage == "" {
		defaultLanguage = defaults.DefaultLanguage
	}
	supportedLanguages := req.SiteSupportedLanguages
	if len(supportedLanguages) == 0 {
		supportedLanguages = defaults.SupportedLanguages
	}

	normalizedDefault, normalizedSupported, err := lang.EnsureDefault(defaultLanguage, supportedLanguages)
	if err != nil {
		return fmt.Errorf("invalid language configuration: %w", err)
	}

	encodedSupported, err := lang.EncodeList(normalizedSupported)
	if err != nil {
		return fmt.Errorf("failed to encode supported languages: %w", err)
	}

	if err := s.settingRepo.Set(settingKeySiteDefaultLanguage, normalizedDefault); err != nil {
		return err
	}

	if err := s.settingRepo.Set(settingKeySiteSupportedLanguages, encodedSupported); err != nil {
		return err
	}

	return nil
}

func (s *SetupService) GetSiteSettings(defaults models.SiteSettings) (models.SiteSettings, error) {
	if s.settingRepo == nil {
		return defaults, nil
	}

	var err error
	result := defaults

	if value, getErr := s.getSettingValue(settingKeySiteName); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.Name = value
	}

	if value, getErr := s.getSettingValue(settingKeySiteDescription); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.Description = value
	}

	if value, getErr := s.getSettingValue(settingKeySiteURL); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.URL = value
	}

	if value, getErr := s.getSettingValue(settingKeySiteFavicon); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.Favicon = value
	}

	result.FaviconType = models.DetectFaviconType(result.Favicon)

	if value, getErr := s.getSettingValue(settingKeySiteLogo); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.Logo = value
	}

	if value, getErr := s.getSettingValue(settingKeyTagRetentionHours); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		if hours, parseErr := strconv.Atoi(value); parseErr == nil && hours > 0 {
			result.UnusedTagRetentionHours = hours
		} else if parseErr != nil {
			err = parseErr
		}
	}

	defaultLang, supported, langErr := s.loadLanguageSettings(result.DefaultLanguage, result.SupportedLanguages)
	if langErr != nil {
		err = errors.Join(err, langErr)
	}

	result.DefaultLanguage = defaultLang
	result.SupportedLanguages = supported

	return result, err
}

func (s *SetupService) ReplaceFavicon(file *multipart.FileHeader) (string, string, error) {
	if s.settingRepo == nil {
		return "", "", errors.New("setting repository not configured")
	}

	if s.uploadService == nil {
		return "", "", errors.New("upload service not configured")
	}

	if file == nil {
		return "", "", &InvalidFaviconError{Reason: "no favicon provided"}
	}

	if err := s.uploadService.ValidateImage(file); err != nil {
		return "", "", &InvalidFaviconError{Reason: err.Error()}
	}

	existingValue := ""
	hadExisting := false

	if value, err := s.getSettingValue(settingKeySiteFavicon); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", err
		}
	} else {
		existingValue = strings.TrimSpace(value)
		hadExisting = true
	}

	newURL, _, err := s.uploadService.UploadImage(file, "")
	if err != nil {
		return "", "", err
	}

	if err := s.settingRepo.Set(settingKeySiteFavicon, newURL); err != nil {
		s.uploadService.DeleteImage(newURL)
		return "", "", err
	}

	if existingValue != "" && s.uploadService.IsManagedURL(existingValue) {
		if err := s.uploadService.DeleteImage(existingValue); err != nil && !errors.Is(err, os.ErrNotExist) {
			if hadExisting {
				s.settingRepo.Set(settingKeySiteFavicon, existingValue)
			} else {
				s.settingRepo.Delete(settingKeySiteFavicon)
			}
			s.uploadService.DeleteImage(newURL)
			return "", "", err
		}
	}

	faviconType := models.DetectFaviconType(newURL)
	return newURL, faviconType, nil
}

func (s *SetupService) ReplaceLogo(file *multipart.FileHeader) (string, error) {
	if s.settingRepo == nil {
		return "", errors.New("setting repository not configured")
	}

	if s.uploadService == nil {
		return "", errors.New("upload service not configured")
	}

	if file == nil {
		return "", &InvalidLogoError{Reason: "no logo provided"}
	}

	if err := s.uploadService.ValidateImage(file); err != nil {
		return "", &InvalidLogoError{Reason: err.Error()}
	}

	existingValue := ""
	hadExisting := false

	if value, err := s.getSettingValue(settingKeySiteLogo); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	} else {
		existingValue = strings.TrimSpace(value)
		hadExisting = true
	}

	newURL, _, err := s.uploadService.UploadImage(file, "")
	if err != nil {
		return "", err
	}

	if err := s.settingRepo.Set(settingKeySiteLogo, newURL); err != nil {
		s.uploadService.DeleteImage(newURL)
		return "", err
	}

	if existingValue != "" && s.uploadService.IsManagedURL(existingValue) {
		if err := s.uploadService.DeleteImage(existingValue); err != nil && !errors.Is(err, os.ErrNotExist) {
			if hadExisting {
				s.settingRepo.Set(settingKeySiteLogo, existingValue)
			} else {
				s.settingRepo.Delete(settingKeySiteLogo)
			}
			s.uploadService.DeleteImage(newURL)
			return "", err
		}
	}

	return newURL, nil
}

func (s *SetupService) UpdateSiteSettings(req models.UpdateSiteSettingsRequest, defaults models.SiteSettings) error {
	if s.settingRepo == nil {
		return errors.New("setting repository not configured")
	}

	updates := map[string]string{
		settingKeySiteName:          strings.TrimSpace(req.Name),
		settingKeySiteDescription:   strings.TrimSpace(req.Description),
		settingKeySiteURL:           strings.TrimSpace(req.URL),
		settingKeySiteFavicon:       strings.TrimSpace(req.Favicon),
		settingKeySiteLogo:          strings.TrimSpace(req.Logo),
		settingKeyTagRetentionHours: strconv.Itoa(req.UnusedTagRetentionHours),
	}

	for key, value := range updates {
		if value == "" {
			if err := s.settingRepo.Delete(key); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			continue
		}

		if err := s.settingRepo.Set(key, value); err != nil {
			return err
		}
	}

	defaultLanguage := strings.TrimSpace(req.DefaultLanguage)
	if defaultLanguage == "" {
		defaultLanguage = defaults.DefaultLanguage
	}
	supportedLanguages := req.SupportedLanguages
	if len(supportedLanguages) == 0 {
		supportedLanguages = defaults.SupportedLanguages
	}

	normalizedDefault, normalizedSupported, err := lang.EnsureDefault(defaultLanguage, supportedLanguages)
	if err != nil {
		return fmt.Errorf("invalid language configuration: %w", err)
	}

	encodedSupported, err := lang.EncodeList(normalizedSupported)
	if err != nil {
		return fmt.Errorf("failed to encode supported languages: %w", err)
	}

	if err := s.settingRepo.Set(settingKeySiteDefaultLanguage, normalizedDefault); err != nil {
		return err
	}

	if err := s.settingRepo.Set(settingKeySiteSupportedLanguages, encodedSupported); err != nil {
		return err
	}

	return nil
}

func (s *SetupService) loadLanguageSettings(defaultLanguage string, supported []string) (string, []string, error) {
	resolvedDefault := strings.TrimSpace(defaultLanguage)
	if resolvedDefault == "" {
		resolvedDefault = lang.Default
	}

	resolvedSupported := supported
	if len(resolvedSupported) == 0 {
		resolvedSupported = []string{resolvedDefault}
	}

	var combinedErr error

	if s.settingRepo != nil {
		if value, err := s.getSettingValue(settingKeySiteDefaultLanguage); err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				combinedErr = errors.Join(combinedErr, err)
			}
		} else if trimmed := strings.TrimSpace(value); trimmed != "" {
			resolvedDefault = trimmed
		}

		if value, err := s.getSettingValue(settingKeySiteSupportedLanguages); err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				combinedErr = errors.Join(combinedErr, err)
			}
		} else if trimmed := strings.TrimSpace(value); trimmed != "" {
			parsed, parseErr := lang.DecodeList(trimmed)
			if parseErr != nil {
				combinedErr = errors.Join(combinedErr, parseErr)
			} else if len(parsed) > 0 {
				resolvedSupported = parsed
			}
		}
	}

	normalizedDefault, normalizedSupported, err := lang.EnsureDefault(resolvedDefault, resolvedSupported)
	if err != nil {
		combinedErr = errors.Join(combinedErr, err)
	}

	if len(normalizedSupported) == 0 {
		normalizedSupported = []string{normalizedDefault}
	}

	return normalizedDefault, normalizedSupported, combinedErr
}

func (s *SetupService) GetSiteLanguages(defaultLanguage string, supported []string) (string, []string, error) {
	return s.loadLanguageSettings(defaultLanguage, supported)
}

func (s *SetupService) getSettingValue(key string) (string, error) {
	if s.settingRepo == nil {
		return "", gorm.ErrRecordNotFound
	}

	setting, err := s.settingRepo.Get(key)
	if err != nil {
		return "", err
	}

	return setting.Value, nil
}

const (
	settingKeySetupComplete          = "setup.completed"
	settingKeySiteName               = "site.name"
	settingKeySiteDescription        = "site.description"
	settingKeySiteURL                = "site.url"
	settingKeySiteFavicon            = "site.favicon"
	settingKeySiteLogo               = "site.logo"
	settingKeyTagRetentionHours      = blogservice.SettingKeyTagRetentionHours
	settingKeySiteDefaultLanguage    = "site.default_language"
	settingKeySiteSupportedLanguages = "site.supported_languages"
)
