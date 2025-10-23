package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"

	"gorm.io/gorm"
)

const (
	// SettingKeyAdvertising stores advertising configuration in the settings repository.
	SettingKeyAdvertising = "site.advertising"
)

// RenderedAdvertising represents provider specific markup prepared for rendering in templates.
// Markup entries are returned as raw HTML strings that should be marked safe at the template layer.
type RenderedAdvertising struct {
	Enabled      bool
	HeadSnippets []string
	Placements   map[string][]string
}

type AdvertisingProvider interface {
	Key() string
	Metadata() models.AdvertisingProviderMetadata
	Normalize(settings models.AdvertisingSettings) (models.AdvertisingSettings, error)
	Render(settings models.AdvertisingSettings) (RenderedAdvertising, error)
}

type advertisingSecurityDirectiveProvider interface {
	SecurityDirectives(settings models.AdvertisingSettings) models.ContentSecurityPolicyDirectives
}

type AdvertisingService struct {
	settingRepo repository.SettingRepository
	providers   map[string]AdvertisingProvider
}

type AdvertisingValidationError struct {
	Reason string
}

func (e *AdvertisingValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Reason
}

func validationErrorf(format string, args ...interface{}) error {
	return &AdvertisingValidationError{Reason: fmt.Sprintf(format, args...)}
}

func NewAdvertisingService(repo repository.SettingRepository) *AdvertisingService {
	svc := &AdvertisingService{
		settingRepo: repo,
		providers:   make(map[string]AdvertisingProvider),
	}
	svc.registerProvider(newGoogleAdsProvider())
	return svc
}

func (s *AdvertisingService) registerProvider(provider AdvertisingProvider) {
	if provider == nil {
		return
	}
	key := strings.TrimSpace(strings.ToLower(provider.Key()))
	if key == "" {
		return
	}
	s.providers[key] = provider
}

func (s *AdvertisingService) Providers() []models.AdvertisingProviderMetadata {
	if len(s.providers) == 0 {
		return nil
	}
	metadata := make([]models.AdvertisingProviderMetadata, 0, len(s.providers))
	for _, provider := range s.providers {
		metadata = append(metadata, provider.Metadata())
	}
	sort.Slice(metadata, func(i, j int) bool {
		return metadata[i].Key < metadata[j].Key
	})
	return metadata
}

func (s *AdvertisingService) DefaultSettings() models.AdvertisingSettings {
	return models.AdvertisingSettings{
		Enabled:  false,
		Provider: "",
	}
}

func (s *AdvertisingService) GetSettings() (models.AdvertisingSettings, error) {
	defaults := s.DefaultSettings()
	if s.settingRepo == nil {
		return defaults, nil
	}

	stored, err := s.settingRepo.Get(SettingKeyAdvertising)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaults, nil
		}
		return defaults, err
	}

	if strings.TrimSpace(stored.Value) == "" {
		return defaults, nil
	}

	var settings models.AdvertisingSettings
	if err := json.Unmarshal([]byte(stored.Value), &settings); err != nil {
		return defaults, fmt.Errorf("failed to decode advertising settings: %w", err)
	}

	s.ensureProviderDefaults(&settings)

	return settings, nil
}

func (s *AdvertisingService) ensureProviderDefaults(settings *models.AdvertisingSettings) {
	if settings == nil {
		return
	}
	settings.Provider = strings.TrimSpace(strings.ToLower(settings.Provider))
	if settings.Provider == googleAdsProviderKey {
		if settings.GoogleAds == nil {
			settings.GoogleAds = &models.GoogleAdsSettings{}
		}
		if settings.GoogleAds.Slots == nil {
			settings.GoogleAds.Slots = []models.GoogleAdsSlot{}
		}
	}
}

func (s *AdvertisingService) UpdateSettings(req models.UpdateAdvertisingSettingsRequest) (models.AdvertisingSettings, error) {
	settings := models.AdvertisingSettings{
		Enabled:  req.Enabled,
		Provider: strings.TrimSpace(strings.ToLower(req.Provider)),
		GoogleAds: func(cfg *models.GoogleAdsSettings) *models.GoogleAdsSettings {
			if cfg == nil {
				return nil
			}
			cloned := *cfg
			if cloned.Slots == nil {
				cloned.Slots = []models.GoogleAdsSlot{}
			}
			return &cloned
		}(req.GoogleAds),
	}

	if settings.Provider == "" {
		if settings.GoogleAds != nil {
			settings.GoogleAds = nil
		}
		if settings.Enabled {
			return models.AdvertisingSettings{}, validationErrorf("advertising provider is required when advertising is enabled")
		}
		return s.persist(settings)
	}

	provider, ok := s.providers[settings.Provider]
	if !ok {
		if settings.Enabled {
			return models.AdvertisingSettings{}, validationErrorf("unsupported advertising provider: %s", settings.Provider)
		}
		settings.Provider = ""
		settings.GoogleAds = nil
		return s.persist(settings)
	}

	normalized, err := provider.Normalize(settings)
	if err != nil {
		return models.AdvertisingSettings{}, err
	}

	return s.persist(normalized)
}

func (s *AdvertisingService) persist(settings models.AdvertisingSettings) (models.AdvertisingSettings, error) {
	if settings.Provider == googleAdsProviderKey && settings.GoogleAds != nil && settings.GoogleAds.Slots == nil {
		settings.GoogleAds.Slots = []models.GoogleAdsSlot{}
	}

	if s.settingRepo == nil {
		return settings, nil
	}

	payload, err := json.Marshal(settings)
	if err != nil {
		return settings, fmt.Errorf("failed to encode advertising settings: %w", err)
	}

	if err := s.settingRepo.Set(SettingKeyAdvertising, string(payload)); err != nil {
		return settings, err
	}

	return settings, nil
}

func (s *AdvertisingService) RenderActive() (RenderedAdvertising, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return RenderedAdvertising{}, err
	}
	if !settings.Enabled {
		return RenderedAdvertising{Enabled: false}, nil
	}

	provider, ok := s.providers[settings.Provider]
	if !ok {
		return RenderedAdvertising{Enabled: false}, nil
	}

	rendered, err := provider.Render(settings)
	if err != nil {
		return RenderedAdvertising{}, err
	}

	rendered.Enabled = settings.Enabled && rendered.Enabled
	return rendered, nil
}

// ContentSecurityPolicyDirectives returns additional CSP directives required by the active advertising provider.
// When advertising is disabled or no provider-specific directives are defined an empty directive map is returned.
func (s *AdvertisingService) ContentSecurityPolicyDirectives() models.ContentSecurityPolicyDirectives {
	directives := make(models.ContentSecurityPolicyDirectives)

	if s == nil {
		return directives
	}

	settings, err := s.GetSettings()
	if err != nil {
		return directives
	}

	if !settings.Enabled {
		return directives
	}

	provider, ok := s.providers[settings.Provider]
	if !ok {
		return directives
	}

	securityProvider, ok := provider.(advertisingSecurityDirectiveProvider)
	if !ok {
		return directives
	}

	if providerDirectives := securityProvider.SecurityDirectives(settings); providerDirectives != nil {
		for directive, sources := range providerDirectives {
			if len(sources) == 0 {
				continue
			}
			copyOfSources := make([]string, len(sources))
			copy(copyOfSources, sources)
			directives[directive] = copyOfSources
		}
	}

	return directives
}
