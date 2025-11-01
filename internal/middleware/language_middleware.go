package middleware

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/lang"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

type languageResolver struct {
	setup             *service.SetupService
	fallbackDefault   string
	fallbackSupported []string

	mu            sync.RWMutex
	cachedDefault string
	cachedList    []string
	lastLoaded    time.Time
}

func newLanguageResolver(cfg *config.Config, setup *service.SetupService) *languageResolver {
	defaultLanguage := lang.Default
	supported := []string{defaultLanguage}

	if cfg != nil {
		if normalized := strings.TrimSpace(cfg.DefaultLanguage); normalized != "" {
			defaultLanguage = normalized
		}
		if len(cfg.SupportedLanguages) > 0 {
			supported = append([]string(nil), cfg.SupportedLanguages...)
		} else {
			supported = []string{defaultLanguage}
		}
	}

	return &languageResolver{
		setup:             setup,
		fallbackDefault:   defaultLanguage,
		fallbackSupported: supported,
		cachedDefault:     defaultLanguage,
		cachedList:        append([]string(nil), supported...),
		lastLoaded:        time.Now(),
	}
}

func (r *languageResolver) languages() (string, []string) {
	r.mu.RLock()
	defaultLang := r.cachedDefault
	supported := append([]string(nil), r.cachedList...)
	last := r.lastLoaded
	r.mu.RUnlock()

	if len(supported) == 0 {
		supported = append([]string(nil), r.fallbackSupported...)
		if len(supported) == 0 {
			supported = []string{r.fallbackDefault}
		}
		defaultLang = r.fallbackDefault
	}

	if r.setup == nil {
		return defaultLang, supported
	}

	if time.Since(last) < time.Minute && len(supported) > 0 {
		return defaultLang, supported
	}

	fallbackDefault := defaultLang
	fallbackSupported := supported
	if fallbackDefault == "" {
		fallbackDefault = r.fallbackDefault
	}
	if len(fallbackSupported) == 0 {
		fallbackSupported = r.fallbackSupported
	}

	resolvedDefault, resolvedSupported, err := r.setup.GetSiteLanguages(fallbackDefault, fallbackSupported)
	if err != nil {
		logger.Error(err, "Failed to resolve site languages", nil)
		resolvedDefault = fallbackDefault
		resolvedSupported = fallbackSupported
	}
	if len(resolvedSupported) == 0 {
		resolvedSupported = []string{resolvedDefault}
	}

	r.mu.Lock()
	r.cachedDefault = resolvedDefault
	r.cachedList = append([]string(nil), resolvedSupported...)
	r.lastLoaded = time.Now()
	r.mu.Unlock()

	return resolvedDefault, resolvedSupported
}

func (r *languageResolver) resolve(explicit, acceptHeader string) (string, []string) {
	defaultLang, supported := r.languages()

	if normalized, err := lang.Normalize(explicit); err == nil {
		if containsLanguage(supported, normalized) {
			return normalized, supported
		}
		if base := matchBaseLanguage(normalized, supported); base != "" {
			return base, supported
		}
	}

	preferences := parseAcceptLanguage(acceptHeader)
	for _, pref := range preferences {
		if containsLanguage(supported, pref) {
			return pref, supported
		}
		if base := matchBaseLanguage(pref, supported); base != "" {
			return base, supported
		}
	}

	if defaultLang == "" && len(supported) > 0 {
		defaultLang = supported[0]
	}
	if defaultLang == "" {
		defaultLang = lang.Default
	}

	return defaultLang, supported
}

func containsLanguage(list []string, code string) bool {
	for _, item := range list {
		if strings.EqualFold(item, code) {
			return true
		}
	}
	return false
}

func matchBaseLanguage(code string, list []string) string {
	parts := strings.SplitN(code, "-", 2)
	if len(parts) == 0 {
		return ""
	}
	base := parts[0]
	for _, candidate := range list {
		if candidate == base {
			return candidate
		}
		if strings.HasPrefix(candidate, base+"-") {
			return candidate
		}
	}
	return ""
}

func parseAcceptLanguage(header string) []string {
	if strings.TrimSpace(header) == "" {
		return nil
	}

	type entry struct {
		code   string
		weight float64
		index  int
	}

	parts := strings.Split(header, ",")
	entries := make([]entry, 0, len(parts))

	for idx, part := range parts {
		segment := strings.TrimSpace(part)
		if segment == "" {
			continue
		}

		weight := 1.0
		code := segment

		if semi := strings.Index(segment, ";"); semi != -1 {
			code = strings.TrimSpace(segment[:semi])
			params := strings.Split(segment[semi+1:], ";")
			for _, param := range params {
				kv := strings.SplitN(strings.TrimSpace(param), "=", 2)
				if len(kv) != 2 {
					continue
				}
				if kv[0] == "q" {
					if parsed, err := strconv.ParseFloat(kv[1], 64); err == nil {
						weight = parsed
					}
				}
			}
		}

		normalized, err := lang.Normalize(code)
		if err != nil {
			continue
		}

		entries = append(entries, entry{code: normalized, weight: weight, index: idx})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].weight == entries[j].weight {
			return entries[i].index < entries[j].index
		}
		return entries[i].weight > entries[j].weight
	})

	result := make([]string, 0, len(entries))
	for _, item := range entries {
		result = append(result, item.code)
	}
	return result
}

// LanguageNegotiationMiddleware resolves the most appropriate language for the
// incoming request using an explicit "lang" query parameter or the
// Accept-Language header. The resolved language and the list of supported
// languages are stored in the request context under the keys "language" and
// "supported_languages" respectively.
func LanguageNegotiationMiddleware(cfg *config.Config, setupService *service.SetupService) gin.HandlerFunc {
	resolver := newLanguageResolver(cfg, setupService)

	return func(c *gin.Context) {
		language, supported := resolver.resolve(c.Query("lang"), c.GetHeader("Accept-Language"))
		if len(supported) == 0 {
			supported = []string{language}
		}

		c.Set("language", language)
		c.Set("supported_languages", supported)
		c.Writer.Header().Set("Content-Language", language)

		c.Next()
	}
}
