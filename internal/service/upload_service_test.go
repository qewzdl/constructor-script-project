package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUploadVideoSuccess(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	content := buildTestMP4(t, buildMvhdVersion0Payload(1000, 45*1000))
	file := createMultipartFile(t, "intro.mp4", content)

	result, err := svc.UploadVideo(context.Background(), file, "Course Intro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Video.URL != "/uploads/course-intro.mp4" {
		t.Fatalf("unexpected url: %s", result.Video.URL)
	}
	if result.Video.Filename != "course-intro.mp4" {
		t.Fatalf("unexpected filename: %s", result.Video.Filename)
	}
	if result.Duration != 45*time.Second {
		t.Fatalf("unexpected duration: %v", result.Duration)
	}
	if result.Subtitle != nil {
		t.Fatalf("expected subtitles to be nil when generator is not configured")
	}

	stored := filepath.Join(uploadDir, result.Video.Filename)
	if _, err := os.Stat(stored); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestUploadVideoInvalidExtension(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	content := []byte("not an mp4")
	file := createMultipartFile(t, "intro.txt", content)

	if _, err := svc.UploadVideo(context.Background(), file, "Course Intro"); err == nil {
		t.Fatal("expected error for invalid file type")
	}
}

func TestUploadVideoDurationFailureDoesNotDelete(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	// Create a file with a valid extension but invalid MP4 content.
	content := []byte("invalid data")
	file := createMultipartFile(t, "intro.mp4", content)

	result, err := svc.UploadVideo(context.Background(), file, "Course Intro")
	if err != nil {
		t.Fatalf("unexpected error for media with unreadable duration: %v", err)
	}

	if result.Duration != 0 {
		t.Fatalf("expected zero duration when parsing fails, got %v", result.Duration)
	}

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		t.Fatalf("failed to read upload dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected uploaded file to remain, found %d entries", len(entries))
	}
}

func TestUseExistingVideoSuccess(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	content := buildTestMP4(t, buildMvhdVersion0Payload(1000, 45*1000))
	file := createMultipartFile(t, "intro.mp4", content)

	uploaded, err := svc.UploadVideo(context.Background(), file, "Course Intro")
	if err != nil {
		t.Fatalf("unexpected error uploading seed video: %v", err)
	}

	result, err := svc.UseExistingVideo(context.Background(), uploaded.Video.URL, "Existing Intro")
	if err != nil {
		t.Fatalf("unexpected error using existing video: %v", err)
	}

	if result.Video.URL != uploaded.Video.URL {
		t.Fatalf("unexpected url: %s", result.Video.URL)
	}
	if result.Video.Filename != uploaded.Video.Filename {
		t.Fatalf("unexpected filename: %s", result.Video.Filename)
	}
	if result.Duration != uploaded.Duration {
		t.Fatalf("unexpected duration: %v", result.Duration)
	}

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		t.Fatalf("failed to read upload dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected existing video to be reused without duplication, found %d entries", len(entries))
	}
}

func TestUploadVideoWithSubtitles(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	subtitleData := []byte("WEBVTT\n\n00:00:00.000 --> 00:00:05.000\nHello world\n")
	generator := &stubSubtitleGenerator{
		result: &SubtitleResult{Data: subtitleData, Format: SubtitleFormatVTT},
	}
	svc.SetSubtitleGenerator(generator)

	content := buildTestMP4(t, buildMvhdVersion0Payload(1000, 45*1000))
	file := createMultipartFile(t, "intro.mp4", content)

	result, err := svc.UploadVideo(context.Background(), file, "Course Intro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Subtitle == nil {
		t.Fatalf("expected subtitle metadata to be present")
	}
	if result.Subtitle.URL != "/uploads/course-intro-subtitles.vtt" {
		t.Fatalf("unexpected subtitle url: %s", result.Subtitle.URL)
	}
	if result.Subtitle.Type != string(UploadCategoryFile) {
		t.Fatalf("unexpected subtitle type: %s", result.Subtitle.Type)
	}

	stored := filepath.Join(uploadDir, result.Subtitle.Filename)
	data, err := os.ReadFile(stored)
	if err != nil {
		t.Fatalf("expected subtitle file to exist: %v", err)
	}
	if !bytes.Equal(data, subtitleData) {
		t.Fatalf("unexpected subtitle content: %q", string(data))
	}

	if len(generator.requests) != 1 {
		t.Fatalf("expected subtitle generator to be called once, got %d", len(generator.requests))
	}
	if generator.requests[0].SourcePath == "" {
		t.Fatal("expected subtitle source path to be provided")
	}
}

func TestUploadVideoRespectsSubtitleConfiguration(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	generator := &stubSubtitleGenerator{result: &SubtitleResult{Data: []byte("WEBVTT\n\n00:00:00.000 --> 00:00:03.000\nConfigured\n"), Format: SubtitleFormatVTT}}
	manager := NewSubtitleManager("openai")
	if err := manager.Register("openai", generator); err != nil {
		t.Fatalf("failed to register subtitle generator: %v", err)
	}
	svc.UseSubtitleManager(manager)

	temperature := float32(0.25)
	svc.ConfigureSubtitleGeneration(SubtitleGenerationConfig{
		Provider:      "openai",
		PreferredName: "Lesson transcript",
		Language:      "en",
		Prompt:        "Course context",
		Temperature:   &temperature,
	})

	content := buildTestMP4(t, buildMvhdVersion0Payload(1000, 30*1000))
	file := createMultipartFile(t, "lesson.mp4", content)

	result, err := svc.UploadVideo(context.Background(), file, "Lesson")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Subtitle == nil {
		t.Fatal("expected subtitle metadata")
	}

	if len(generator.requests) != 1 {
		t.Fatalf("expected single subtitle generation request, got %d", len(generator.requests))
	}

	request := generator.requests[0]
	if request.Provider != "openai" {
		t.Fatalf("unexpected provider: %q", request.Provider)
	}
	if request.Language != "en" {
		t.Fatalf("unexpected language: %q", request.Language)
	}
	if request.Prompt != "Course context" {
		t.Fatalf("unexpected prompt: %q", request.Prompt)
	}
	if request.Temperature == nil || *request.Temperature != temperature {
		t.Fatalf("unexpected temperature: %v", request.Temperature)
	}
	if request.PreferredName != "Lesson transcript" {
		t.Fatalf("unexpected preferred name: %q", request.PreferredName)
	}

	if !strings.Contains(result.Subtitle.Filename, "lesson-transcript") {
		t.Fatalf("expected subtitle filename to include preferred name, got %q", result.Subtitle.Filename)
	}
}

func TestUploadVideoSubtitleFailure(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)
	svc.SetSubtitleGenerator(&stubSubtitleGenerator{err: errors.New("transcription failed")})

	content := buildTestMP4(t, buildMvhdVersion0Payload(1000, 45*1000))
	file := createMultipartFile(t, "intro.mp4", content)

	if _, err := svc.UploadVideo(context.Background(), file, "Course Intro"); err == nil {
		t.Fatal("expected subtitle generation error")
	}

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		t.Fatalf("failed to read upload dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected upload directory to be empty when subtitles fail, found %d entries", len(entries))
	}
}

func TestSaveSubtitle(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	content := []byte("WEBVTT\n\n00:00:00.000 --> 00:00:02.000\nHi there\n")
	info, err := svc.SaveSubtitle("lesson.mp4", content, "Lesson subtitles")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected subtitle metadata")
	}
	if info.URL == "" {
		t.Fatal("expected subtitle url to be set")
	}
	stored := filepath.Join(uploadDir, info.Filename)
	data, err := os.ReadFile(stored)
	if err != nil {
		t.Fatalf("expected subtitle file to exist: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Fatalf("unexpected subtitle contents: %q", string(data))
	}
}

func TestSaveSubtitleEmptyContent(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	if _, err := svc.SaveSubtitle("lesson.mp4", []byte("   \n\t"), "Lesson subtitles"); !errors.Is(err, ErrSubtitleContentEmpty) {
		t.Fatalf("expected ErrSubtitleContentEmpty, got %v", err)
	}
}

func TestUploadFileSuccess(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	content := []byte("project plan")
	file := createMultipartFile(t, "plan.pdf", content)

	info, err := svc.Upload(file, "Project Plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Type != string(UploadCategoryFile) {
		t.Fatalf("unexpected type: %s", info.Type)
	}

	if info.Filename != "project-plan.pdf" {
		t.Fatalf("unexpected filename: %s", info.Filename)
	}

	stored := filepath.Join(uploadDir, info.Filename)
	if _, err := os.Stat(stored); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestListUploadsIncludesTypes(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	image := createMultipartFile(t, "cover.jpg", []byte("image"))
	if _, err := svc.Upload(image, "Cover"); err != nil {
		t.Fatalf("unexpected error uploading image: %v", err)
	}

	document := createMultipartFile(t, "notes.txt", []byte("notes"))
	if _, err := svc.Upload(document, "Meeting Notes"); err != nil {
		t.Fatalf("unexpected error uploading document: %v", err)
	}

	uploads, err := svc.ListUploads()
	if err != nil {
		t.Fatalf("unexpected error listing uploads: %v", err)
	}

	if len(uploads) != 2 {
		t.Fatalf("expected 2 uploads, got %d", len(uploads))
	}

	types := map[string]bool{}
	for _, upload := range uploads {
		types[upload.Type] = true
	}

	if !types[string(UploadCategoryImage)] {
		t.Fatalf("expected image upload to be present")
	}
	if !types[string(UploadCategoryFile)] {
		t.Fatalf("expected file upload to be present")
	}
}

func TestRenameUploadKeepsType(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	document := createMultipartFile(t, "notes.txt", []byte("notes"))
	info, err := svc.Upload(document, "Notes")
	if err != nil {
		t.Fatalf("unexpected error uploading file: %v", err)
	}

	renamed, err := svc.RenameUpload(info.URL, "Project Notes")
	if err != nil {
		t.Fatalf("unexpected error renaming upload: %v", err)
	}

	if renamed.Type != string(UploadCategoryFile) {
		t.Fatalf("unexpected type after rename: %s", renamed.Type)
	}

	if !strings.HasSuffix(renamed.Filename, ".txt") {
		t.Fatalf("expected filename to preserve extension, got %s", renamed.Filename)
	}
}

func TestDeleteUploadRemovesFile(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	document := createMultipartFile(t, "notes.txt", []byte("notes"))
	info, err := svc.Upload(document, "Notes")
	if err != nil {
		t.Fatalf("unexpected error uploading file: %v", err)
	}

	if err := svc.DeleteUpload(info.URL); err != nil {
		t.Fatalf("unexpected error deleting upload: %v", err)
	}

	stored := filepath.Join(uploadDir, info.Filename)
	if _, err := os.Stat(stored); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be removed, got %v", err)
	}

	if err := svc.DeleteUpload(info.URL); !errors.Is(err, ErrUploadNotFound) {
		t.Fatalf("expected ErrUploadNotFound on subsequent delete, got %v", err)
	}
}

type stubSubtitleGenerator struct {
	result   *SubtitleResult
	err      error
	requests []SubtitleGenerationRequest
}

func (s *stubSubtitleGenerator) Generate(ctx context.Context, request SubtitleGenerationRequest) (*SubtitleResult, error) {
	s.requests = append(s.requests, request)
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}
func createMultipartFile(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write file content: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if err := req.ParseMultipartForm(int64(body.Len())); err != nil {
		t.Fatalf("failed to parse multipart form: %v", err)
	}

	files := req.MultipartForm.File["file"]
	if len(files) == 0 {
		t.Fatalf("expected multipart file to be available")
	}

	return files[0]
}

// Helper utilities to build minimal MP4 payloads for exercising the upload
// workflow without relying on external fixtures.
func buildTestMP4(t *testing.T, mvhdPayload []byte) []byte {
	t.Helper()
	moov := buildBox("moov", buildBox("mvhd", mvhdPayload))
	ftyp := buildBox("ftyp", []byte("isom"))
	return append(ftyp, moov...)
}

func buildMvhdVersion0Payload(timescale, duration uint32) []byte {
	payload := make([]byte, 4+16)
	payload[0] = 0
	binary.BigEndian.PutUint32(payload[12:16], timescale)
	binary.BigEndian.PutUint32(payload[16:20], duration)
	return payload
}

func buildBox(boxType string, payload []byte) []byte {
	if len(boxType) != 4 {
		panic("box type must be 4 characters")
	}
	size := uint32(len(payload) + 8)
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, size)
	data := make([]byte, 0, size)
	data = append(data, header...)
	data = append(data, boxType...)
	data = append(data, payload...)
	return data
}
