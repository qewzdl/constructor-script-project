package service

import (
	"bytes"
	"encoding/binary"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUploadVideoSuccess(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	content := buildTestMP4(t, buildMvhdVersion0Payload(1000, 45*1000))
	file := createMultipartFile(t, "intro.mp4", content)

	url, filename, duration, err := svc.UploadVideo(file, "Course Intro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if url != "/uploads/course-intro.mp4" {
		t.Fatalf("unexpected url: %s", url)
	}
	if filename != "course-intro.mp4" {
		t.Fatalf("unexpected filename: %s", filename)
	}
	if duration != 45*time.Second {
		t.Fatalf("unexpected duration: %v", duration)
	}

	stored := filepath.Join(uploadDir, filename)
	if _, err := os.Stat(stored); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestUploadVideoInvalidExtension(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	content := []byte("not an mp4")
	file := createMultipartFile(t, "intro.txt", content)

	if _, _, _, err := svc.UploadVideo(file, "Course Intro"); err == nil {
		t.Fatal("expected error for invalid file type")
	}
}

func TestUploadVideoInvalidMediaRemoved(t *testing.T) {
	uploadDir := t.TempDir()
	svc := NewUploadService(uploadDir)

	// Create a file with a valid extension but invalid MP4 content.
	content := []byte("invalid data")
	file := createMultipartFile(t, "intro.mp4", content)

	if _, _, _, err := svc.UploadVideo(file, "Course Intro"); err == nil {
		t.Fatal("expected error for invalid media")
	}

	// The upload should have been removed after the duration parsing failed.
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		t.Fatalf("failed to read upload dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected upload directory to be empty, found %d entries", len(entries))
	}
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
