package service

import "context"

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

// SubtitleGenerator defines behaviour for services capable of generating subtitles
// for a given media file.
type SubtitleGenerator interface {
	Generate(ctx context.Context, sourcePath string) (*SubtitleResult, error)
}
