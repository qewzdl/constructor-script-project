package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
)

var (
	// ErrSetupAlreadyCompleted is returned when setup has already been completed.
	ErrSetupAlreadyCompleted = errors.New("setup already completed")
)

type SetupService struct {
	userRepo    repository.UserRepository
	settingRepo repository.SettingRepository
}

func NewSetupService(userRepo repository.UserRepository, settingRepo repository.SettingRepository) *SetupService {
	return &SetupService{
		userRepo:    userRepo,
		settingRepo: settingRepo,
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

func (s *SetupService) CompleteSetup(req models.SetupRequest) (*models.User, error) {
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
		Role:     "admin",
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	if s.settingRepo != nil {
		if err := s.saveSiteSettings(req); err != nil {
			return nil, err
		}

		if err := s.settingRepo.Set(settingKeySetupComplete, "true"); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *SetupService) saveSiteSettings(req models.SetupRequest) error {
	settings := map[string]string{
		settingKeySiteName:        req.SiteName,
		settingKeySiteDescription: req.SiteDescription,
		settingKeySiteURL:         req.SiteURL,
		settingKeySiteFavicon:     req.SiteFavicon,
		settingKeySiteLogo:        req.SiteLogo,
	}

	for key, value := range settings {
		if value == "" {
			continue
		}

		if err := s.settingRepo.Set(key, value); err != nil {
			return err
		}
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

	if value, getErr := s.getSettingValue(settingKeySiteLogo); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.Logo = value
	}

	return result, err
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
	settingKeySetupComplete   = "setup.completed"
	settingKeySiteName        = "site.name"
	settingKeySiteDescription = "site.description"
	settingKeySiteURL         = "site.url"
	settingKeySiteFavicon     = "site.favicon"
	settingKeySiteLogo        = "site.logo"
)
