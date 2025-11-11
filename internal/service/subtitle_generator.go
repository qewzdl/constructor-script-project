package service

import (
	"context"
	"errors"
)

// SubtitleFormat represents the container/encoding for generated subtitles.
type SubtitleFormat string

const (
	// SubtitleFormatVTT denotes WebVTT subtitle output.
	SubtitleFormatVTT SubtitleFormat = "vtt"
)

// SubtitleResult contains the generated subtitle payload and related metadata.
type SubtitleResult struct {
	Data     []byte
	Format   SubtitleFormat
	Language string
	Name     string
}

// SubtitleGenerationRequest describes the parameters passed to a subtitle provider.
type SubtitleGenerationRequest struct {
	// SourcePath is the absolute path of the media file to be transcribed.
	SourcePath string
	// Provider is the desired subtitle provider identifier. When empty the
	// default provider configured on the manager will be used.
	Provider string
	// PreferredName indicates the friendly name to assign to the resulting
	// subtitle track. Providers may choose to honour or ignore this value.
	PreferredName string
	// Language is the ISO language code to use for transcription. Providers
	// may fall back to their default language if the field is empty.
	Language string
	// Prompt allows callers to influence the subtitle generation process
	// with additional context supported by the provider.
	Prompt string
	// Temperature enables the caller to override the sampling temperature
	// used by the provider, when supported. A nil value means the provider
	// should use its default temperature.
	Temperature *float32
}

// SubtitleGenerator defines behaviour for services capable of generating
// subtitles for a given media file.
type SubtitleGenerator interface {
	Generate(ctx context.Context, request SubtitleGenerationRequest) (*SubtitleResult, error)
}

// ErrSubtitleProviderNotConfigured is returned when subtitle generation is
// requested but no provider has been registered.
var ErrSubtitleProviderNotConfigured = errors.New("subtitle provider is not configured")
