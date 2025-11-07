package service

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"constructor-script-backend/internal/authorization"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/payments/stripe"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/lang"
	blogservice "constructor-script-backend/plugins/blog/service"
	languageservice "constructor-script-backend/plugins/language/service"
)

var (
	// ErrSetupAlreadyCompleted is returned when setup has already been completed.
	ErrSetupAlreadyCompleted = errors.New("setup already completed")
	currencyCodePattern      = regexp.MustCompile(`^[a-z]{3}$`)
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
	language      *languageservice.LanguageService
}

func NewSetupService(userRepo repository.UserRepository, settingRepo repository.SettingRepository, uploadService *UploadService, languageService *languageservice.LanguageService) *SetupService {
	return &SetupService{
		userRepo:      userRepo,
		settingRepo:   settingRepo,
		uploadService: uploadService,
		language:      languageService,
	}
}

// SetLanguageService updates the language service dependency used by the setup service.
func (s *SetupService) SetLanguageService(languageService *languageservice.LanguageService) {
	if s == nil {
		return
	}
	s.language = languageService
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

	if err := s.updateSiteLanguages(defaultLanguage, supportedLanguages); err != nil {
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

	if value, getErr := s.getSettingValue(settingKeyStripeSecretKey); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.StripeSecretKey = strings.TrimSpace(value)
	}

	if value, getErr := s.getSettingValue(settingKeyStripePublishableKey); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.StripePublishableKey = strings.TrimSpace(value)
	}

	if value, getErr := s.getSettingValue(settingKeyStripeWebhookSecret); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.StripeWebhookSecret = strings.TrimSpace(value)
	}

	if value, getErr := s.getSettingValue(settingKeyCourseCheckoutSuccessURL); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.CourseCheckoutSuccessURL = strings.TrimSpace(value)
	}

	if value, getErr := s.getSettingValue(settingKeyCourseCheckoutCancelURL); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.CourseCheckoutCancelURL = strings.TrimSpace(value)
	}

	if value, getErr := s.getSettingValue(settingKeyCourseCheckoutCurrency); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.CourseCheckoutCurrency = strings.ToLower(strings.TrimSpace(value))
	}

	defaultLang, supported, langErr := s.resolveSiteLanguages(result.DefaultLanguage, result.SupportedLanguages)
	if langErr != nil {
		err = errors.Join(err, langErr)
	}

	result.DefaultLanguage = defaultLang
	result.SupportedLanguages = supported

	if strings.TrimSpace(result.StripeSecretKey) == "" {
		result.StripeSecretKey = strings.TrimSpace(defaults.StripeSecretKey)
	}
	if strings.TrimSpace(result.StripePublishableKey) == "" {
		result.StripePublishableKey = strings.TrimSpace(defaults.StripePublishableKey)
	}
	if strings.TrimSpace(result.StripeWebhookSecret) == "" {
		result.StripeWebhookSecret = strings.TrimSpace(defaults.StripeWebhookSecret)
	}
	if strings.TrimSpace(result.CourseCheckoutSuccessURL) == "" {
		result.CourseCheckoutSuccessURL = strings.TrimSpace(defaults.CourseCheckoutSuccessURL)
	}
	if strings.TrimSpace(result.CourseCheckoutCancelURL) == "" {
		result.CourseCheckoutCancelURL = strings.TrimSpace(defaults.CourseCheckoutCancelURL)
	}
	if strings.TrimSpace(result.CourseCheckoutCurrency) == "" {
		result.CourseCheckoutCurrency = strings.ToLower(strings.TrimSpace(defaults.CourseCheckoutCurrency))
	}

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

	normalizeCheckoutURL := func(value, label string) (string, error) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return "", nil
		}
		parsed, err := url.Parse(trimmed)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return "", fmt.Errorf("invalid %s URL: must be absolute and include scheme and host", label)
		}
		return trimmed, nil
	}

	successURL, err := normalizeCheckoutURL(req.CourseCheckoutSuccessURL, "course checkout success")
	if err != nil {
		return err
	}
	cancelURL, err := normalizeCheckoutURL(req.CourseCheckoutCancelURL, "course checkout cancel")
	if err != nil {
		return err
	}

	currency := strings.ToLower(strings.TrimSpace(req.CourseCheckoutCurrency))
	if currency != "" && !currencyCodePattern.MatchString(currency) {
		return fmt.Errorf("invalid course checkout currency: must be a three-letter ISO code")
	}

	stripeSecret := strings.TrimSpace(req.StripeSecretKey)
	if stripeSecret != "" && !stripe.IsSecretKey(stripeSecret) {
		return fmt.Errorf(
			"invalid Stripe secret key: must start with %q or %q",
			stripe.SecretKeyPrefixStandard,
			stripe.SecretKeyPrefixRestricted,
		)
	}

	stripePublish := strings.TrimSpace(req.StripePublishableKey)
	if stripePublish != "" && !stripe.IsPublishableKey(stripePublish) {
		return fmt.Errorf(
			"invalid Stripe publishable key: must start with %q",
			stripe.PublishableKeyPrefix,
		)
	}

	stripeWebhook := strings.TrimSpace(req.StripeWebhookSecret)
	if stripeWebhook != "" && !stripe.IsWebhookSecret(stripeWebhook) {
		return fmt.Errorf(
			"invalid Stripe webhook secret: must start with %q",
			stripe.WebhookSecretPrefix,
		)
	}

	updates := map[string]string{
		settingKeySiteName:                 strings.TrimSpace(req.Name),
		settingKeySiteDescription:          strings.TrimSpace(req.Description),
		settingKeySiteURL:                  strings.TrimSpace(req.URL),
		settingKeySiteFavicon:              strings.TrimSpace(req.Favicon),
		settingKeySiteLogo:                 strings.TrimSpace(req.Logo),
		settingKeyTagRetentionHours:        strconv.Itoa(req.UnusedTagRetentionHours),
		settingKeyStripeSecretKey:          stripeSecret,
		settingKeyStripePublishableKey:     stripePublish,
		settingKeyStripeWebhookSecret:      stripeWebhook,
		settingKeyCourseCheckoutSuccessURL: successURL,
		settingKeyCourseCheckoutCancelURL:  cancelURL,
		settingKeyCourseCheckoutCurrency:   currency,
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

	if err := s.updateSiteLanguages(defaultLanguage, supportedLanguages); err != nil {
		return err
	}

	return nil
}

func (s *SetupService) resolveSiteLanguages(defaultLanguage string, supported []string) (string, []string, error) {
	if s.language != nil {
		return s.language.Resolve(defaultLanguage, supported)
	}

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

func (s *SetupService) updateSiteLanguages(defaultLanguage string, supported []string) error {
	if s.language != nil {
		return s.language.Update(defaultLanguage, supported)
	}

	if s.settingRepo == nil {
		return nil
	}

	normalizedDefault, normalizedSupported, err := lang.EnsureDefault(defaultLanguage, supported)
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
	settingKeySetupComplete            = "setup.completed"
	settingKeySiteName                 = "site.name"
	settingKeySiteDescription          = "site.description"
	settingKeySiteURL                  = "site.url"
	settingKeySiteFavicon              = "site.favicon"
	settingKeySiteLogo                 = "site.logo"
	settingKeyTagRetentionHours        = blogservice.SettingKeyTagRetentionHours
	settingKeySiteDefaultLanguage      = "site.default_language"
	settingKeySiteSupportedLanguages   = "site.supported_languages"
	settingKeyStripeSecretKey          = "payments.stripe.secret_key"
	settingKeyStripePublishableKey     = "payments.stripe.publishable_key"
	settingKeyStripeWebhookSecret      = "payments.stripe.webhook_secret"
	settingKeyCourseCheckoutSuccessURL = "courses.checkout.success_url"
	settingKeyCourseCheckoutCancelURL  = "courses.checkout.cancel_url"
	settingKeyCourseCheckoutCurrency   = "courses.checkout.currency"
)
