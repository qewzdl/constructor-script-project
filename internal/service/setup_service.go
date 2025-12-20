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
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/lang"
	"constructor-script-backend/pkg/logger"
	blogservice "constructor-script-backend/plugins/blog/service"
	languageservice "constructor-script-backend/plugins/language/service"
)

var (
	// ErrSetupAlreadyCompleted is returned when setup has already been completed.
	ErrSetupAlreadyCompleted = errors.New("setup already completed")
	currencyCodePattern      = regexp.MustCompile(`^[a-z]{3}$`)
	emailPattern             = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

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
	db            *gorm.DB
}

func NewSetupService(userRepo repository.UserRepository, settingRepo repository.SettingRepository, uploadService *UploadService, languageService *languageservice.LanguageService) *SetupService {
	return &SetupService{
		userRepo:      userRepo,
		settingRepo:   settingRepo,
		uploadService: uploadService,
		language:      languageService,
	}
}

// SetDB sets the database connection for managing setup progress
func (s *SetupService) SetDB(db *gorm.DB) {
	if s == nil {
		return
	}
	s.db = db
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

	// Validate setup request
	if err := s.validateSetupRequest(req); err != nil {
		return nil, err
	}

	count, err := s.userRepo.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to check user count: %w", err)
	}

	if count > 0 {
		return nil, ErrSetupAlreadyCompleted
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Username: strings.TrimSpace(req.AdminUsername),
		Email:    strings.ToLower(strings.TrimSpace(req.AdminEmail)),
		Password: string(hashedPassword),
		Role:     authorization.RoleAdmin,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	if s.settingRepo != nil {
		if err := s.saveSiteSettings(req, defaults); err != nil {
			return nil, fmt.Errorf("failed to save site settings: %w", err)
		}

		if err := s.settingRepo.Set(settingKeySetupComplete, "true"); err != nil {
			return nil, fmt.Errorf("failed to mark setup as complete: %w", err)
		}
	}

	logger.Info("Setup completed successfully", map[string]interface{}{
		"admin_username": user.Username,
		"admin_email":    user.Email,
	})

	return user, nil
}

func (s *SetupService) saveSiteSettings(req models.SetupRequest, defaults models.SiteSettings) error {
	settings := map[string]string{
		settingKeySiteName:          req.SiteName,
		settingKeySiteDescription:   req.SiteDescription,
		settingKeySiteURL:           req.SiteURL,
		settingKeySiteFavicon:       req.SiteFavicon,
		settingKeySiteLogo:          req.SiteLogo,
		settingKeySiteContactEmail:  strings.TrimSpace(req.SiteContactEmail),
		settingKeySiteFooterText:    strings.TrimSpace(req.SiteFooterText),
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

	if defaults.Subtitles.Temperature != nil {
		value := *defaults.Subtitles.Temperature
		result.Subtitles.Temperature = &value
	}

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
	if value, getErr := s.getSettingValue(settingKeySiteContactEmail); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.ContactEmail = strings.TrimSpace(value)
	}
	if value, getErr := s.getSettingValue(settingKeySiteFooterText); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = getErr
		}
	} else if value != "" {
		result.FooterText = strings.TrimSpace(value)
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

	subtitleSettings, subtitleErr := s.GetSubtitleSettings(defaults.Subtitles)
	if subtitleErr != nil {
		err = errors.Join(err, subtitleErr)
	}
	result.Subtitles = subtitleSettings

	return result, err
}

func (s *SetupService) GetSubtitleSettings(defaults models.SubtitleSettings) (models.SubtitleSettings, error) {
	result := defaults
	if defaults.Temperature != nil {
		value := *defaults.Temperature
		result.Temperature = &value
	}

	if s.settingRepo == nil {
		normalizeSubtitleSettings(&result)
		return result, nil
	}

	var err error

	if value, getErr := s.getSettingValue(settingKeySubtitlesEnabled); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = errors.Join(err, getErr)
		}
	} else if trimmed := strings.TrimSpace(value); trimmed != "" {
		if parsed, parseErr := strconv.ParseBool(trimmed); parseErr != nil {
			err = errors.Join(err, parseErr)
		} else {
			result.Enabled = parsed
		}
	}

	if value, getErr := s.getSettingValue(settingKeySubtitlesProvider); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = errors.Join(err, getErr)
		}
	} else if trimmed := strings.TrimSpace(value); trimmed != "" {
		result.Provider = trimmed
	}

	if value, getErr := s.getSettingValue(settingKeySubtitlesPreferredName); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = errors.Join(err, getErr)
		}
	} else if trimmed := strings.TrimSpace(value); trimmed != "" {
		result.PreferredName = trimmed
	}

	if value, getErr := s.getSettingValue(settingKeySubtitlesLanguage); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = errors.Join(err, getErr)
		}
	} else if trimmed := strings.TrimSpace(value); trimmed != "" {
		result.Language = trimmed
	}

	if value, getErr := s.getSettingValue(settingKeySubtitlesPrompt); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = errors.Join(err, getErr)
		}
	} else if trimmed := strings.TrimSpace(value); trimmed != "" {
		result.Prompt = trimmed
	}

	if value, getErr := s.getSettingValue(settingKeySubtitlesTemperature); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = errors.Join(err, getErr)
		}
	} else if trimmed := strings.TrimSpace(value); trimmed != "" {
		if parsed, parseErr := strconv.ParseFloat(trimmed, 32); parseErr != nil {
			err = errors.Join(err, parseErr)
		} else {
			temp := float32(parsed)
			result.Temperature = &temp
		}
	}

	if value, getErr := s.getSettingValue(settingKeySubtitlesOpenAIModel); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = errors.Join(err, getErr)
		}
	} else if trimmed := strings.TrimSpace(value); trimmed != "" {
		result.OpenAIModel = trimmed
	}

	if value, getErr := s.getSettingValue(settingKeySubtitlesOpenAIAPIKey); getErr != nil {
		if !errors.Is(getErr, gorm.ErrRecordNotFound) {
			err = errors.Join(err, getErr)
		}
	} else if trimmed := strings.TrimSpace(value); trimmed != "" {
		result.OpenAIAPIKey = trimmed
	}

	normalizeSubtitleSettings(&result)

	return result, err
}

func (s *SetupService) updateSubtitleSettings(req models.UpdateSubtitleSettingsRequest, defaults models.SubtitleSettings) error {
	if s.settingRepo == nil {
		return errors.New("setting repository not configured")
	}

	provider := normalizeSubtitleProvider(req.Provider)
	if provider == "" {
		provider = normalizeSubtitleProvider(defaults.Provider)
	}
	if provider == "" {
		provider = "openai"
	}

	switch provider {
	case "openai":
	case "default", "":
		provider = "openai"
	default:
		return fmt.Errorf("unsupported subtitle provider: %s", req.Provider)
	}

	currentKey := strings.TrimSpace(defaults.OpenAIAPIKey)
	if value, err := s.getSettingValue(settingKeySubtitlesOpenAIAPIKey); err == nil {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			currentKey = trimmed
		}
	}

	requestedKey := strings.TrimSpace(req.OpenAIAPIKey)
	finalKey := requestedKey
	if finalKey == "" {
		finalKey = currentKey
	}

	if finalKey == "" {
		finalKey = strings.TrimSpace(defaults.OpenAIAPIKey)
	}

	if req.Enabled && provider == "openai" && finalKey == "" {
		return fmt.Errorf("OpenAI API key is required when enabling subtitle generation")
	}

	subtitleUpdates := map[string]string{
		settingKeySubtitlesEnabled:       strconv.FormatBool(req.Enabled),
		settingKeySubtitlesProvider:      provider,
		settingKeySubtitlesPreferredName: strings.TrimSpace(req.PreferredName),
		settingKeySubtitlesLanguage:      strings.TrimSpace(req.Language),
		settingKeySubtitlesPrompt:        strings.TrimSpace(req.Prompt),
		settingKeySubtitlesOpenAIModel:   strings.TrimSpace(req.OpenAIModel),
	}

	if req.Temperature != nil {
		subtitleUpdates[settingKeySubtitlesTemperature] = strconv.FormatFloat(float64(*req.Temperature), 'f', -1, 32)
	} else {
		subtitleUpdates[settingKeySubtitlesTemperature] = ""
	}

	for key, value := range subtitleUpdates {
		if value == "" {
			if key == settingKeySubtitlesEnabled || key == settingKeySubtitlesProvider {
				continue
			}
			if err := s.settingRepo.Delete(key); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			continue
		}

		if err := s.settingRepo.Set(key, value); err != nil {
			return err
		}
	}

	switch {
	case requestedKey != "":
		if err := s.settingRepo.Set(settingKeySubtitlesOpenAIAPIKey, requestedKey); err != nil {
			return err
		}
	case currentKey != "" && req.Enabled:
		// Preserve existing key when still enabled and no new value provided.
	default:
		if err := s.settingRepo.Delete(settingKeySubtitlesOpenAIAPIKey); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	return nil
}

func (s *SetupService) refreshSubtitleConfiguration(defaults models.SubtitleSettings) {
	if s == nil || s.uploadService == nil {
		return
	}

	settings, err := s.GetSubtitleSettings(defaults)
	if err != nil {
		logger.Error(err, "Failed to reload subtitle settings", nil)
		return
	}

	ConfigureUploadSubtitles(s.uploadService, settings)
}

func normalizeSubtitleSettings(settings *models.SubtitleSettings) {
	if settings == nil {
		return
	}

	settings.Provider = normalizeSubtitleProvider(settings.Provider)
	if settings.Provider == "" {
		settings.Provider = "openai"
	}
	settings.PreferredName = strings.TrimSpace(settings.PreferredName)
	settings.Language = strings.TrimSpace(settings.Language)
	settings.Prompt = strings.TrimSpace(settings.Prompt)
	settings.OpenAIModel = strings.TrimSpace(settings.OpenAIModel)
	settings.OpenAIAPIKey = strings.TrimSpace(settings.OpenAIAPIKey)
	if settings.Temperature != nil {
		value := *settings.Temperature
		settings.Temperature = &value
	}
}

func normalizeSubtitleProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
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

	currentStripeSecret := s.currentSettingValue(settingKeyStripeSecretKey, defaults.StripeSecretKey)
	currentStripePublishable := s.currentSettingValue(settingKeyStripePublishableKey, defaults.StripePublishableKey)
	currentStripeWebhook := s.currentSettingValue(settingKeyStripeWebhookSecret, defaults.StripeWebhookSecret)

	stripeSecret, updateStripeSecret := normalizeCredentialInput(req.StripeSecretKey, currentStripeSecret)
	stripePublish, updateStripePublish := normalizeCredentialInput(req.StripePublishableKey, currentStripePublishable)
	stripeWebhook, updateStripeWebhook := normalizeCredentialInput(req.StripeWebhookSecret, currentStripeWebhook)

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
		return &ValidationError{Field: "course_checkout_success_url", Message: err.Error()}
	}
	cancelURL, err := normalizeCheckoutURL(req.CourseCheckoutCancelURL, "course checkout cancel")
	if err != nil {
		return &ValidationError{Field: "course_checkout_cancel_url", Message: err.Error()}
	}

	currency := strings.ToLower(strings.TrimSpace(req.CourseCheckoutCurrency))
	if currency != "" && !currencyCodePattern.MatchString(currency) {
		return &ValidationError{
			Field:   "course_checkout_currency",
			Message: "invalid course checkout currency: must be a three-letter ISO code",
		}
	}

	contactEmail := strings.TrimSpace(req.ContactEmail)
	if contactEmail != "" && !emailPattern.MatchString(contactEmail) {
		return &ValidationError{
			Field:   "contact_email",
			Message: "invalid contact email address",
		}
	}

	updates := map[string]string{
		settingKeySiteName:                 strings.TrimSpace(req.Name),
		settingKeySiteDescription:          strings.TrimSpace(req.Description),
		settingKeySiteURL:                  strings.TrimSpace(req.URL),
		settingKeySiteFavicon:              strings.TrimSpace(req.Favicon),
		settingKeySiteLogo:                 strings.TrimSpace(req.Logo),
		settingKeySiteContactEmail:         contactEmail,
		settingKeySiteFooterText:           strings.TrimSpace(req.FooterText),
		settingKeyTagRetentionHours:        strconv.Itoa(req.UnusedTagRetentionHours),
		settingKeyCourseCheckoutSuccessURL: successURL,
		settingKeyCourseCheckoutCancelURL:  cancelURL,
		settingKeyCourseCheckoutCurrency:   currency,
	}

	if updateStripeSecret {
		updates[settingKeyStripeSecretKey] = stripeSecret
	}
	if updateStripePublish {
		updates[settingKeyStripePublishableKey] = stripePublish
	}
	if updateStripeWebhook {
		updates[settingKeyStripeWebhookSecret] = stripeWebhook
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

	if req.Subtitles != nil {
		if err := s.updateSubtitleSettings(*req.Subtitles, defaults.Subtitles); err != nil {
			return err
		}
		s.refreshSubtitleConfiguration(defaults.Subtitles)
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

func normalizeCredentialInput(input, current string) (string, bool) {
	trimmedInput := strings.TrimSpace(input)
	currentValue := strings.TrimSpace(current)

	switch {
	case trimmedInput == "":
		return "", true
	case isMaskedCredential(trimmedInput):
		return currentValue, false
	case currentValue != "" && trimmedInput == currentValue:
		return currentValue, false
	default:
		return trimmedInput, true
	}
}

func (s *SetupService) currentSettingValue(key string, fallback string) string {
	value := strings.TrimSpace(fallback)

	if stored, err := s.getSettingValue(key); err == nil {
		if trimmed := strings.TrimSpace(stored); trimmed != "" {
			value = trimmed
		}
	}

	return value
}

func isMaskedCredential(value string) bool {
	if value == "" {
		return false
	}

	for _, r := range value {
		switch r {
		case '*', '•', '·', '●':
		default:
			return false
		}
	}

	return true
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

// validateSetupRequest validates the setup request data
func (s *SetupService) validateSetupRequest(req models.SetupRequest) error {
	// Validate username
	username := strings.TrimSpace(req.AdminUsername)
	if len(username) < 3 {
		return &ValidationError{Field: "admin_username", Message: "must be at least 3 characters"}
	}
	if len(username) > 50 {
		return &ValidationError{Field: "admin_username", Message: "must not exceed 50 characters"}
	}

	// Validate email
	email := strings.TrimSpace(req.AdminEmail)
	if email == "" {
		return &ValidationError{Field: "admin_email", Message: "is required"}
	}
	if !emailPattern.MatchString(email) {
		return &ValidationError{Field: "admin_email", Message: "is invalid"}
	}

	// Validate password strength
	if len(req.AdminPassword) < 8 {
		return &ValidationError{Field: "admin_password", Message: "must be at least 8 characters"}
	}
	if len(req.AdminPassword) > 128 {
		return &ValidationError{Field: "admin_password", Message: "must not exceed 128 characters"}
	}

	// Check password complexity
	hasUpper := false
	hasLower := false
	hasDigit := false
	for _, char := range req.AdminPassword {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return &ValidationError{
			Field:   "admin_password",
			Message: "must contain at least one uppercase letter, one lowercase letter, and one digit",
		}
	}

	// Validate site name
	siteName := strings.TrimSpace(req.SiteName)
	if siteName == "" {
		return &ValidationError{Field: "site_name", Message: "is required"}
	}
	if len(siteName) > 255 {
		return &ValidationError{Field: "site_name", Message: "must not exceed 255 characters"}
	}

	// Validate site URL if provided
	if req.SiteURL != "" {
		siteURL := strings.TrimSpace(req.SiteURL)
		parsedURL, err := url.Parse(siteURL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			return &ValidationError{Field: "site_url", Message: "must be a valid HTTP or HTTPS URL"}
		}
	}

	// Validate language settings
	if req.SiteDefaultLanguage != "" {
		if _, err := lang.Normalize(req.SiteDefaultLanguage); err != nil {
			return &ValidationError{Field: "site_default_language", Message: "is not a valid language code"}
		}
	}

	return nil
}

const (
	settingKeySetupComplete            = "setup.completed"
	settingKeySiteName                 = "site.name"
	settingKeySiteDescription          = "site.description"
	settingKeySiteURL                  = "site.url"
	settingKeySiteFavicon              = "site.favicon"
	settingKeySiteLogo                 = "site.logo"
	settingKeySiteContactEmail         = "site.contact_email"
	settingKeySiteFooterText           = "site.footer_text"
	settingKeyTagRetentionHours        = blogservice.SettingKeyTagRetentionHours
	settingKeySiteDefaultLanguage      = "site.default_language"
	settingKeySiteSupportedLanguages   = "site.supported_languages"
	settingKeyStripeSecretKey          = "payments.stripe.secret_key"
	settingKeyStripePublishableKey     = "payments.stripe.publishable_key"
	settingKeyStripeWebhookSecret      = "payments.stripe.webhook_secret"
	settingKeyCourseCheckoutSuccessURL = "courses.checkout.success_url"
	settingKeyCourseCheckoutCancelURL  = "courses.checkout.cancel_url"
	settingKeyCourseCheckoutCurrency   = "courses.checkout.currency"
	settingKeySubtitlesEnabled         = "media.subtitles.enabled"
	settingKeySubtitlesProvider        = "media.subtitles.provider"
	settingKeySubtitlesPreferredName   = "media.subtitles.preferred_name"
	settingKeySubtitlesLanguage        = "media.subtitles.language"
	settingKeySubtitlesPrompt          = "media.subtitles.prompt"
	settingKeySubtitlesTemperature     = "media.subtitles.temperature"
	settingKeySubtitlesOpenAIModel     = "media.subtitles.openai_model"
	settingKeySubtitlesOpenAIAPIKey    = "media.subtitles.openai_api_key"
)

// GetSetupProgress retrieves the current setup progress
func (s *SetupService) GetSetupProgress() (*models.SetupProgress, error) {
	if s.db == nil {
		return nil, errors.New("database not configured")
	}

	var progress models.SetupProgress
	err := s.db.First(&progress).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create initial progress record
			progress = models.SetupProgress{
				CurrentStep: string(models.SetupStepSiteInfo),
			}
			if err := s.db.Create(&progress).Error; err != nil {
				return nil, fmt.Errorf("failed to create setup progress: %w", err)
			}
			return &progress, nil
		}
		return nil, fmt.Errorf("failed to get setup progress: %w", err)
	}

	return &progress, nil
}

// SaveStepData saves data for a specific setup step
func (s *SetupService) SaveStepData(req models.SetupStepRequest) (*models.SetupProgress, error) {
	if s.db == nil {
		return nil, errors.New("database not configured")
	}

	// Validate the step request
	if err := req.ValidateStep(); err != nil {
		return nil, err
	}

	progress, err := s.GetSetupProgress()
	if err != nil {
		return nil, err
	}

	step := models.SetupStep(req.Step)

	// Save data based on step
	switch step {
	case models.SetupStepSiteInfo:
		data := req.ToSiteInfoData()
		data.Name = strings.TrimSpace(data.Name)
		data.Description = strings.TrimSpace(data.Description)
		data.URL = strings.TrimSpace(data.URL)
		data.Favicon = strings.TrimSpace(data.Favicon)
		data.Logo = strings.TrimSpace(data.Logo)

		progress.SiteInfo = data
		progress.MarkStepComplete(step)

	case models.SetupStepAdmin:
		data := req.ToAdminData()

		// Hash password for storage
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}

		data.Username = strings.TrimSpace(data.Username)
		data.Email = strings.ToLower(strings.TrimSpace(data.Email))
		data.Password = string(hashedPassword)

		progress.Admin = data
		progress.MarkStepComplete(step)

	case models.SetupStepLanguages:
		data := req.ToLanguagesData()
		data.DefaultLanguage = strings.TrimSpace(data.DefaultLanguage)

		progress.Languages = data
		progress.MarkStepComplete(step)

	default:
		return nil, fmt.Errorf("invalid setup step: %s", req.Step)
	}

	if err := s.db.Save(progress).Error; err != nil {
		return nil, fmt.Errorf("failed to save setup progress: %w", err)
	}

	return progress, nil
}

// CompleteStepwiseSetup finalizes the setup after all steps are completed
func (s *SetupService) CompleteStepwiseSetup(defaults models.SiteSettings) (*models.User, error) {
	if s.userRepo == nil {
		return nil, errors.New("user repository not configured")
	}

	progress, err := s.GetSetupProgress()
	if err != nil {
		return nil, fmt.Errorf("failed to get setup progress: %w", err)
	}

	if !progress.AllStepsComplete() {
		missingSteps := []string{}
		if !progress.SiteInfoComplete {
			missingSteps = append(missingSteps, "site_info")
		}
		if !progress.AdminComplete {
			missingSteps = append(missingSteps, "admin")
		}
		if !progress.LanguagesComplete {
			missingSteps = append(missingSteps, "languages")
		}
		return nil, fmt.Errorf("incomplete setup steps: %v", strings.Join(missingSteps, ", "))
	}

	// Check if setup already completed
	count, err := s.userRepo.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to check user count: %w", err)
	}
	if count > 0 {
		return nil, ErrSetupAlreadyCompleted
	}

	// Create admin user from progress data
	user := &models.User{
		Username: progress.Admin.Username,
		Email:    progress.Admin.Email,
		Password: progress.Admin.Password, // Already hashed
		Role:     authorization.RoleAdmin,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	// Save site settings
	if s.settingRepo != nil {
		req := models.SetupRequest{
			SiteName:            progress.SiteInfo.Name,
			SiteDescription:     progress.SiteInfo.Description,
			SiteURL:             progress.SiteInfo.URL,
			SiteFavicon:         progress.SiteInfo.Favicon,
			SiteLogo:            progress.SiteInfo.Logo,
			SiteDefaultLanguage: progress.Languages.DefaultLanguage,
		}

		// Parse supported languages
		if progress.Languages.SupportedLanguages != "" {
			req.SiteSupportedLanguages = strings.Split(progress.Languages.SupportedLanguages, ",")
		}

		if err := s.saveSiteSettings(req, defaults); err != nil {
			return nil, fmt.Errorf("failed to save site settings: %w", err)
		}

		if err := s.settingRepo.Set(settingKeySetupComplete, "true"); err != nil {
			return nil, fmt.Errorf("failed to mark setup as complete: %w", err)
		}
	}

	// Clean up progress record
	if s.db != nil {
		s.db.Delete(progress)
	}

	logger.Info("Stepwise setup completed successfully", map[string]interface{}{
		"admin_username": user.Username,
		"admin_email":    user.Email,
	})

	return user, nil
}

// ResetSetupProgress resets the setup progress (useful for testing or re-setup)
func (s *SetupService) ResetSetupProgress() error {
	if s.db == nil {
		return errors.New("database not configured")
	}

	return s.db.Where("1 = 1").Delete(&models.SetupProgress{}).Error
}
