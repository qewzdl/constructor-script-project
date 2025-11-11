package service

import (
	"strings"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/pkg/logger"
)

// ConfigureUploadSubtitles updates the upload service with the provided subtitle settings.
func ConfigureUploadSubtitles(uploadService *UploadService, settings models.SubtitleSettings) {
	if uploadService == nil {
		return
	}

	if !settings.Enabled {
		uploadService.UseSubtitleManager(nil)
		uploadService.ConfigureSubtitleGeneration(SubtitleGenerationConfig{})
		return
	}

	provider := strings.ToLower(strings.TrimSpace(settings.Provider))
	manager := NewSubtitleManager(provider)
	temperature := float32(0)
	if settings.Temperature != nil {
		temperature = *settings.Temperature
	}

	switch provider {
	case "", "openai":
		generator, err := NewOpenAISubtitleGenerator(strings.TrimSpace(settings.OpenAIAPIKey), OpenAISubtitleOptions{
			Model:       strings.TrimSpace(settings.OpenAIModel),
			Temperature: temperature,
			Prompt:      strings.TrimSpace(settings.Prompt),
			Language:    strings.TrimSpace(settings.Language),
		})
		if err != nil {
			logger.Error(err, "Failed to initialise subtitle generator", map[string]interface{}{"provider": "openai"})
		} else {
			if registerErr := manager.Register("openai", generator); registerErr != nil {
				logger.Error(registerErr, "Failed to register subtitle provider", map[string]interface{}{"provider": "openai"})
			} else if provider == "" {
				manager.SetDefaultProvider("openai")
			}
		}
	default:
		logger.Warn("Unsupported subtitle provider configured; subtitle generation disabled", map[string]interface{}{"provider": provider})
	}

	if providers := manager.Providers(); len(providers) == 0 {
		uploadService.UseSubtitleManager(nil)
		uploadService.ConfigureSubtitleGeneration(SubtitleGenerationConfig{})
		return
	}

	uploadService.UseSubtitleManager(manager)

	var tempPointer *float32
	if settings.Temperature != nil {
		value := *settings.Temperature
		tempPointer = &value
	}

	uploadService.ConfigureSubtitleGeneration(SubtitleGenerationConfig{
		Provider:      provider,
		PreferredName: strings.TrimSpace(settings.PreferredName),
		Language:      strings.TrimSpace(settings.Language),
		Prompt:        strings.TrimSpace(settings.Prompt),
		Temperature:   tempPointer,
	})
}
