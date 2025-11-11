package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultOpenAITranscriptionEndpoint = "https://api.openai.com/v1/audio/transcriptions"

// OpenAISubtitleOptions controls how subtitles are generated via the OpenAI Whisper API.
type OpenAISubtitleOptions struct {
	Model       string
	Temperature float32
	Prompt      string
	Language    string
	Endpoint    string
	HTTPClient  *http.Client
}

// OpenAISubtitleGenerator implements SubtitleGenerator using the OpenAI Whisper API.
type OpenAISubtitleGenerator struct {
	apiKey      string
	model       string
	temperature float32
	prompt      string
	language    string
	endpoint    string
	client      *http.Client
}

// NewOpenAISubtitleGenerator constructs a generator backed by OpenAI Whisper APIs.
func NewOpenAISubtitleGenerator(apiKey string, opts OpenAISubtitleOptions) (*OpenAISubtitleGenerator, error) {
	trimmedKey := strings.TrimSpace(apiKey)
	if trimmedKey == "" {
		return nil, errors.New("openai api key is required for subtitle generation")
	}

	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = "whisper-1"
	}

	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" {
		endpoint = defaultOpenAITranscriptionEndpoint
	}

	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}

	generator := &OpenAISubtitleGenerator{
		apiKey:      trimmedKey,
		model:       model,
		temperature: opts.Temperature,
		prompt:      strings.TrimSpace(opts.Prompt),
		language:    strings.TrimSpace(opts.Language),
		endpoint:    endpoint,
		client:      client,
	}

	return generator, nil
}

// Generate invokes the OpenAI Whisper transcription endpoint to produce WebVTT subtitles.
func (g *OpenAISubtitleGenerator) Generate(ctx context.Context, request SubtitleGenerationRequest) (*SubtitleResult, error) {
	if g == nil || g.client == nil {
		return nil, errors.New("openai subtitle generator is not configured")
	}

	trimmedPath := strings.TrimSpace(request.SourcePath)
	if trimmedPath == "" {
		return nil, errors.New("source path is required")
	}

	file, err := os.Open(trimmedPath)
	if err != nil {
		return nil, fmt.Errorf("openai: failed to open source file: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(trimmedPath))
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("openai: failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		writer.Close()
		return nil, fmt.Errorf("openai: failed to stream source file: %w", err)
	}

	if err := writer.WriteField("model", g.model); err != nil {
		writer.Close()
		return nil, fmt.Errorf("openai: failed to write model field: %w", err)
	}
	if err := writer.WriteField("response_format", "vtt"); err != nil {
		writer.Close()
		return nil, fmt.Errorf("openai: failed to set response format: %w", err)
	}

	prompt := g.prompt
	if candidate := strings.TrimSpace(request.Prompt); candidate != "" {
		prompt = candidate
	}
	if prompt != "" {
		if err := writer.WriteField("prompt", prompt); err != nil {
			writer.Close()
			return nil, fmt.Errorf("openai: failed to set prompt: %w", err)
		}
	}

	language := g.language
	if candidate := strings.TrimSpace(request.Language); candidate != "" {
		language = candidate
	}
	if language != "" {
		if err := writer.WriteField("language", language); err != nil {
			writer.Close()
			return nil, fmt.Errorf("openai: failed to set language: %w", err)
		}
	}

	temperature := g.temperature
	if request.Temperature != nil {
		temperature = *request.Temperature
	}
	if temperature > 0 {
		if err := writer.WriteField("temperature", strconv.FormatFloat(float64(temperature), 'f', -1, 32)); err != nil {
			writer.Close()
			return nil, fmt.Errorf("openai: failed to set temperature: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("openai: failed to finalise request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint, &body)
	if err != nil {
		return nil, fmt.Errorf("openai: failed to build request: %w", err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+g.apiKey)
	httpRequest.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := g.client.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("openai: transcription request failed: %w", err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("openai: failed to read response: %w", err)
	}

	if response.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = response.Status
		}
		return nil, fmt.Errorf("openai: transcription request returned status %s: %s", response.Status, message)
	}

	text := strings.TrimSpace(string(data))
	if text == "" {
		return nil, errors.New("openai transcription returned an empty response")
	}

	output := []byte(text)
	if !strings.HasSuffix(text, "\n") {
		output = append(output, '\n')
	}

	result := &SubtitleResult{
		Data:     output,
		Format:   SubtitleFormatVTT,
		Language: language,
	}
	if request.PreferredName != "" {
		result.Name = request.PreferredName
	}

	return result, nil
}
