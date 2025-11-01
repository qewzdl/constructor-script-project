package service

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/lang"

	"gorm.io/gorm"
)

const languageCacheTTL = time.Minute

// LanguageService centralises management of the site's language configuration.
// It coordinates fallback values coming from the configuration with values
// persisted in the settings repository while providing transparent caching for
// callers.
type LanguageService struct {
	cfg  *config.Config
	repo repository.SettingRepository

	mu            sync.RWMutex
	cachedDefault string
	cachedList    []string
	lastLoaded    time.Time
}

// NewLanguageService creates a new instance of the language service using the
// supplied configuration and settings repository.
func NewLanguageService(cfg *config.Config, repo repository.SettingRepository) *LanguageService {
	service := &LanguageService{
		cfg:  cfg,
		repo: repo,
	}

	defaultLang, supported := service.defaults()
	service.cachedDefault = defaultLang
	service.cachedList = append([]string(nil), supported...)
	service.lastLoaded = time.Now()

	return service
}

// Defaults returns the configured fallback default language and list of
// supported languages. The values are normalised and guaranteed to include at
// least the default language.
func (s *LanguageService) Defaults() (string, []string) {
	return s.defaults()
}

// Resolve returns the effective default language and list of supported
// languages. The repository is consulted when available; otherwise the
// configured defaults are returned. Results are cached to minimise repository
// traffic. The default language is always the first entry in the returned list.
func (s *LanguageService) Resolve(defaultLanguage string, supported []string) (string, []string, error) {
	fallbackDefault, fallbackSupported := s.fallback(defaultLanguage, supported)

	s.mu.RLock()
	cachedDefault := s.cachedDefault
	cachedSupported := append([]string(nil), s.cachedList...)
	lastLoaded := s.lastLoaded
	s.mu.RUnlock()

	if time.Since(lastLoaded) < languageCacheTTL && len(cachedSupported) > 0 {
		return cachedDefault, cachedSupported, nil
	}

	resolvedDefault := fallbackDefault
	resolvedSupported := fallbackSupported
	var combinedErr error

	if s.repo != nil {
		if setting, err := s.repo.Get(settingKeySiteDefaultLanguage); err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				combinedErr = errors.Join(combinedErr, err)
			}
		} else if trimmed := strings.TrimSpace(setting.Value); trimmed != "" {
			resolvedDefault = trimmed
		}

		if setting, err := s.repo.Get(settingKeySiteSupportedLanguages); err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				combinedErr = errors.Join(combinedErr, err)
			}
		} else if trimmed := strings.TrimSpace(setting.Value); trimmed != "" {
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
		normalizedDefault = fallbackDefault
		normalizedSupported = append([]string(nil), fallbackSupported...)
	}

	if len(normalizedSupported) == 0 {
		normalizedSupported = []string{normalizedDefault}
	}

	s.mu.Lock()
	s.cachedDefault = normalizedDefault
	s.cachedList = append([]string(nil), normalizedSupported...)
	s.lastLoaded = time.Now()
	s.mu.Unlock()

	return normalizedDefault, normalizedSupported, combinedErr
}

// Update persists the supplied language configuration and refreshes the cache.
func (s *LanguageService) Update(defaultLanguage string, supported []string) error {
	normalizedDefault, normalizedSupported, err := lang.EnsureDefault(defaultLanguage, supported)
	if err != nil {
		return fmt.Errorf("invalid language configuration: %w", err)
	}

	if len(normalizedSupported) == 0 {
		normalizedSupported = []string{normalizedDefault}
	}

	if s.repo != nil {
		encoded, encodeErr := lang.EncodeList(normalizedSupported)
		if encodeErr != nil {
			return fmt.Errorf("failed to encode supported languages: %w", encodeErr)
		}

		if err := s.repo.Set(settingKeySiteDefaultLanguage, normalizedDefault); err != nil {
			return err
		}
		if err := s.repo.Set(settingKeySiteSupportedLanguages, encoded); err != nil {
			return err
		}
	}

	s.mu.Lock()
	s.cachedDefault = normalizedDefault
	s.cachedList = append([]string(nil), normalizedSupported...)
	s.lastLoaded = time.Now()
	s.mu.Unlock()

	return nil
}

func (s *LanguageService) defaults() (string, []string) {
	fallbackDefault := strings.TrimSpace(lang.Default)
	fallbackSupported := []string{fallbackDefault}

	if s.cfg != nil {
		if normalized, err := lang.Normalize(s.cfg.DefaultLanguage); err == nil && normalized != "" {
			fallbackDefault = normalized
		}

		if len(s.cfg.SupportedLanguages) > 0 {
			if normalized, err := lang.NormalizeList(s.cfg.SupportedLanguages); err == nil && len(normalized) > 0 {
				fallbackSupported = ensureIncludesDefault(fallbackDefault, normalized)
			}
		} else {
			fallbackSupported = []string{fallbackDefault}
		}
	}

	return fallbackDefault, ensureIncludesDefault(fallbackDefault, fallbackSupported)
}

func (s *LanguageService) fallback(defaultLanguage string, supported []string) (string, []string) {
	candidateDefault := strings.TrimSpace(defaultLanguage)
	if candidateDefault == "" {
		defaultFallback, _ := s.defaults()
		candidateDefault = defaultFallback
	}

	normalizedDefault, err := lang.Normalize(candidateDefault)
	if err != nil || normalizedDefault == "" {
		normalizedDefault, _ = s.defaults()
	}

	candidateSupported := supported
	if len(candidateSupported) == 0 {
		_, supportedFallback := s.defaults()
		candidateSupported = supportedFallback
	}

	normalizedSupported, err := lang.NormalizeList(candidateSupported)
	if err != nil || len(normalizedSupported) == 0 {
		_, normalizedSupported = s.defaults()
	}

	return normalizedDefault, ensureIncludesDefault(normalizedDefault, normalizedSupported)
}

func ensureIncludesDefault(defaultLanguage string, supported []string) []string {
	result := make([]string, 0, len(supported)+1)
	seen := make(map[string]struct{}, len(supported)+1)

	result = append(result, defaultLanguage)
	seen[defaultLanguage] = struct{}{}

	for _, code := range supported {
		if strings.TrimSpace(code) == "" {
			continue
		}
		normalized, err := lang.Normalize(code)
		if err != nil {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	return result
}
