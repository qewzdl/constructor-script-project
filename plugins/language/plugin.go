package language

import (
	"fmt"

	"constructor-script-backend/internal/plugin/host"
	"constructor-script-backend/internal/plugin/registry"
	pluginruntime "constructor-script-backend/internal/plugin/runtime"
	languageservice "constructor-script-backend/plugins/language/service"
)

func init() {
	registry.Register("language", NewFeature)
}

type Feature struct {
	host host.Host
}

func NewFeature(h host.Host) (pluginruntime.Feature, error) {
	if h == nil {
		return nil, fmt.Errorf("host is required")
	}
	return &Feature{host: h}, nil
}

func (f *Feature) Activate() error {
	if f == nil || f.host == nil {
		return fmt.Errorf("feature host is not configured")
	}

	repos := f.host.Repositories()
	services := f.host.CoreServices()

	languageService := services.Language()
	if languageService == nil {
		languageService = languageservice.NewLanguageService(f.host.Config(), repos.Setting())
		services.SetLanguage(languageService)
	}

	if setupService := services.Setup(); setupService != nil {
		setupService.SetLanguageService(languageService)
	}

	if templateHandler := f.host.TemplateHandler(); templateHandler != nil {
		templateHandler.SetLanguageService(languageService)
	}

	if seoHandler := f.host.SEOHandler(); seoHandler != nil {
		seoHandler.SetLanguageService(languageService)
	}

	return nil
}

func (f *Feature) Deactivate() error {
	if f == nil || f.host == nil {
		return nil
	}

	services := f.host.CoreServices()

	if templateHandler := f.host.TemplateHandler(); templateHandler != nil {
		templateHandler.SetLanguageService(nil)
	}

	if seoHandler := f.host.SEOHandler(); seoHandler != nil {
		seoHandler.SetLanguageService(nil)
	}

	if setupService := services.Setup(); setupService != nil {
		setupService.SetLanguageService(nil)
	}

	services.SetLanguage(nil)

	return nil
}
