package service

import (
	"strings"
	"testing"

	"constructor-script-backend/internal/models"
)

func float32Ptr(value float32) *float32 {
	v := value
	return &v
}

func TestSetupService_GetSubtitleSettingsUsesDefaults(t *testing.T) {
	repo := newMemorySettingRepository()
	svc := NewSetupService(nil, repo, nil, nil)
	defaults := models.SubtitleSettings{
		Enabled:       true,
		Provider:      "OpenAI",
		PreferredName: "  Lesson subtitles  ",
		Language:      " EN ",
		Prompt:        "  Describe the lesson  ",
		Temperature:   float32Ptr(0.25),
		OpenAIModel:   " whisper-1 ",
		OpenAIAPIKey:  "sk-test",
	}

	settings, err := svc.GetSubtitleSettings(defaults)
	if err != nil {
		t.Fatalf("GetSubtitleSettings returned error: %v", err)
	}

	if !settings.Enabled {
		t.Fatalf("expected subtitles to remain enabled")
	}
	if settings.Provider != "openai" {
		t.Fatalf("expected provider normalised to openai, got %q", settings.Provider)
	}
	if settings.PreferredName != "Lesson subtitles" {
		t.Fatalf("expected preferred name trimmed, got %q", settings.PreferredName)
	}
	if settings.Language != "EN" {
		t.Fatalf("expected language trimmed, got %q", settings.Language)
	}
	if settings.Prompt != "Describe the lesson" {
		t.Fatalf("expected prompt trimmed, got %q", settings.Prompt)
	}
	if settings.OpenAIModel != "whisper-1" {
		t.Fatalf("expected model trimmed, got %q", settings.OpenAIModel)
	}
	if settings.OpenAIAPIKey != "sk-test" {
		t.Fatalf("expected API key preserved")
	}
	if settings.Temperature == nil || *settings.Temperature != 0.25 {
		t.Fatalf("expected temperature copied, got %v", settings.Temperature)
	}
	if settings.Temperature == defaults.Temperature {
		t.Fatalf("expected temperature pointer to be copied")
	}
}

func TestSetupService_GetSubtitleSettingsOverridesStoredValues(t *testing.T) {
	repo := newMemorySettingRepository()
	svc := NewSetupService(nil, repo, nil, nil)
	defaults := models.SubtitleSettings{Enabled: true, Provider: "openai"}

	repo.Set(settingKeySubtitlesEnabled, "false")
	repo.Set(settingKeySubtitlesProvider, "OpenAI")
	repo.Set(settingKeySubtitlesPreferredName, "Course captions")
	repo.Set(settingKeySubtitlesLanguage, "ru")
	repo.Set(settingKeySubtitlesPrompt, "Provide Russian subtitles")
	repo.Set(settingKeySubtitlesTemperature, "0.75")
	repo.Set(settingKeySubtitlesOpenAIModel, "custom-whisper")
	repo.Set(settingKeySubtitlesOpenAIAPIKey, "sk-live")

	settings, err := svc.GetSubtitleSettings(defaults)
	if err != nil {
		t.Fatalf("GetSubtitleSettings returned error: %v", err)
	}

	if settings.Enabled {
		t.Fatalf("expected stored disabled flag to be applied")
	}
	if settings.Provider != "openai" {
		t.Fatalf("expected provider lower-cased, got %q", settings.Provider)
	}
	if settings.PreferredName != "Course captions" {
		t.Fatalf("unexpected preferred name: %q", settings.PreferredName)
	}
	if settings.Language != "ru" {
		t.Fatalf("unexpected language: %q", settings.Language)
	}
	if settings.Prompt != "Provide Russian subtitles" {
		t.Fatalf("unexpected prompt: %q", settings.Prompt)
	}
	if settings.Temperature == nil || *settings.Temperature != 0.75 {
		t.Fatalf("expected stored temperature, got %v", settings.Temperature)
	}
	if settings.OpenAIModel != "custom-whisper" {
		t.Fatalf("unexpected model: %q", settings.OpenAIModel)
	}
	if settings.OpenAIAPIKey != "sk-live" {
		t.Fatalf("unexpected API key: %q", settings.OpenAIAPIKey)
	}
}

func TestSetupService_UpdateSubtitleSettingsPersistsConfiguration(t *testing.T) {
	repo := newMemorySettingRepository()
	svc := NewSetupService(nil, repo, nil, nil)
	defaults := models.SiteSettings{
		DefaultLanguage:    "en",
		SupportedLanguages: []string{"en"},
		Subtitles:          models.SubtitleSettings{Provider: "openai"},
	}

	req := models.UpdateSiteSettingsRequest{
		Subtitles: &models.UpdateSubtitleSettingsRequest{
			Enabled:       true,
			Provider:      "openai",
			PreferredName: " Lesson output ",
			Language:      " fr ",
			Prompt:        "  Write fluent captions  ",
			Temperature:   float32Ptr(0.4),
			OpenAIModel:   " whisper-1 ",
			OpenAIAPIKey:  "sk-new",
		},
	}

	if err := svc.UpdateSiteSettings(req, defaults); err != nil {
		t.Fatalf("UpdateSiteSettings returned error: %v", err)
	}

	if value := repo.store[settingKeySubtitlesEnabled]; value != "true" {
		t.Fatalf("expected enabled flag stored, got %q", value)
	}
	if value := repo.store[settingKeySubtitlesProvider]; value != "openai" {
		t.Fatalf("expected provider stored as openai, got %q", value)
	}
	if value := repo.store[settingKeySubtitlesPreferredName]; value != "Lesson output" {
		t.Fatalf("unexpected preferred name: %q", value)
	}
	if value := repo.store[settingKeySubtitlesLanguage]; value != "fr" {
		t.Fatalf("unexpected language: %q", value)
	}
	if value := repo.store[settingKeySubtitlesPrompt]; value != "Write fluent captions" {
		t.Fatalf("unexpected prompt: %q", value)
	}
	if value := repo.store[settingKeySubtitlesTemperature]; value != "0.4" {
		t.Fatalf("unexpected temperature: %q", value)
	}
	if value := repo.store[settingKeySubtitlesOpenAIModel]; value != "whisper-1" {
		t.Fatalf("unexpected model: %q", value)
	}
	if value := repo.store[settingKeySubtitlesOpenAIAPIKey]; value != "sk-new" {
		t.Fatalf("unexpected api key: %q", value)
	}

	resolved, err := svc.GetSubtitleSettings(defaults.Subtitles)
	if err != nil {
		t.Fatalf("GetSubtitleSettings returned error: %v", err)
	}
	if !resolved.Enabled || resolved.Language != "fr" || resolved.OpenAIAPIKey != "sk-new" {
		t.Fatalf("expected stored values reflected in GetSubtitleSettings, got %+v", resolved)
	}
}

func TestSetupService_UpdateSubtitleSettingsRequiresAPIKeyWhenEnabling(t *testing.T) {
	repo := newMemorySettingRepository()
	svc := NewSetupService(nil, repo, nil, nil)
	defaults := models.SiteSettings{
		DefaultLanguage:    "en",
		SupportedLanguages: []string{"en"},
		Subtitles:          models.SubtitleSettings{Provider: "openai"},
	}

	err := svc.UpdateSiteSettings(models.UpdateSiteSettingsRequest{
		Subtitles: &models.UpdateSubtitleSettingsRequest{Enabled: true, Provider: "openai"},
	}, defaults)
	if err == nil || !strings.Contains(err.Error(), "OpenAI API key is required") {
		t.Fatalf("expected validation error about missing API key, got %v", err)
	}
}

func TestSetupService_UpdateSubtitleSettingsPreservesStoredKey(t *testing.T) {
	repo := newMemorySettingRepository()
	repo.Set(settingKeySubtitlesOpenAIAPIKey, "sk-existing")
	svc := NewSetupService(nil, repo, nil, nil)
	defaults := models.SiteSettings{
		DefaultLanguage:    "en",
		SupportedLanguages: []string{"en"},
		Subtitles:          models.SubtitleSettings{Provider: "openai"},
	}

	if err := svc.UpdateSiteSettings(models.UpdateSiteSettingsRequest{
		Subtitles: &models.UpdateSubtitleSettingsRequest{Enabled: true, Provider: "openai"},
	}, defaults); err != nil {
		t.Fatalf("expected existing key to allow enabling, got %v", err)
	}

	if repo.store[settingKeySubtitlesOpenAIAPIKey] != "sk-existing" {
		t.Fatalf("expected existing key to be preserved")
	}
}

func TestSetupService_UpdateSubtitleSettingsClearsKeyWhenDisabled(t *testing.T) {
	repo := newMemorySettingRepository()
	repo.Set(settingKeySubtitlesOpenAIAPIKey, "sk-existing")
	svc := NewSetupService(nil, repo, nil, nil)
	defaults := models.SiteSettings{
		DefaultLanguage:    "en",
		SupportedLanguages: []string{"en"},
		Subtitles:          models.SubtitleSettings{Provider: "openai"},
	}

	if err := svc.UpdateSiteSettings(models.UpdateSiteSettingsRequest{
		Subtitles: &models.UpdateSubtitleSettingsRequest{Enabled: false, Provider: "openai"},
	}, defaults); err != nil {
		t.Fatalf("expected disable to succeed, got %v", err)
	}

	if _, ok := repo.store[settingKeySubtitlesOpenAIAPIKey]; ok {
		t.Fatalf("expected API key to be cleared when subtitles disabled")
	}
}
