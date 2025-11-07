package media

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMP4DurationVersion0(t *testing.T) {
	duration := 45 * time.Second
	data := buildTestMP4(t, buildMvhdVersion0Payload(1000, 45*1000))

	got, err := mp4DurationFromReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != duration {
		t.Fatalf("expected duration %v, got %v", duration, got)
	}

	// Ensure the exported helper also works when reading from disk.
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sample.mp4")
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	gotFromFile, err := MP4Duration(filePath)
	if err != nil {
		t.Fatalf("unexpected error from MP4Duration: %v", err)
	}
	if gotFromFile != duration {
		t.Fatalf("expected duration %v, got %v", duration, gotFromFile)
	}
}

func TestMP4DurationVersion1(t *testing.T) {
	duration := 90 * time.Second
	data := buildTestMP4(t, buildMvhdVersion1Payload(1000, 90*1000))

	got, err := mp4DurationFromReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != duration {
		t.Fatalf("expected duration %v, got %v", duration, got)
	}
}

func TestMP4DurationIgnoresEmptyBoxes(t *testing.T) {
	t.Helper()

	mvhd := buildMvhdVersion0Payload(1000, 45*1000)
	moovPayload := append(buildBox("free", nil), buildBox("mvhd", mvhd)...)
	data := append(buildBox("ftyp", []byte("isom")), buildBox("moov", moovPayload)...)

	got, err := mp4DurationFromReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 45 * time.Second
	if got != expected {
		t.Fatalf("expected duration %v, got %v", expected, got)
	}
}

func TestMP4DurationErrors(t *testing.T) {
	t.Run("no moov", func(t *testing.T) {
		data := buildBox("ftyp", []byte("isom"))
		if _, err := mp4DurationFromReader(bytes.NewReader(data)); err == nil {
			t.Fatal("expected error when moov box is missing")
		}
	})

	t.Run("no mvhd", func(t *testing.T) {
		moovPayload := buildBox("trak", []byte("dummy"))
		data := append(buildBox("ftyp", []byte("isom")), buildBox("moov", moovPayload)...)
		if _, err := mp4DurationFromReader(bytes.NewReader(data)); err == nil {
			t.Fatal("expected error when mvhd box is missing")
		}
	})

	t.Run("zero timescale", func(t *testing.T) {
		mvhd := buildMvhdVersion0Payload(0, 100)
		data := buildTestMP4(t, mvhd)
		if _, err := mp4DurationFromReader(bytes.NewReader(data)); err == nil {
			t.Fatal("expected error when timescale is zero")
		}
	})
}

func buildTestMP4(t *testing.T, mvhdPayload []byte) []byte {
	t.Helper()
	moov := buildBox("moov", buildBox("mvhd", mvhdPayload))
	ftyp := buildBox("ftyp", []byte("isom"))
	return append(ftyp, moov...)
}

func buildBox(boxType string, payload []byte) []byte {
	if len(boxType) != 4 {
		panic("box type must be 4 characters")
	}
	payloadLen := 0
	if payload != nil {
		payloadLen = len(payload)
	}
	size := uint32(payloadLen + 8)
	buf := make([]byte, 0, size)
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, size)
	buf = append(buf, header...)
	buf = append(buf, boxType...)
	if payloadLen > 0 {
		buf = append(buf, payload...)
	}
	return buf
}

func buildMvhdVersion0Payload(timescale, duration uint32) []byte {
	payload := make([]byte, 4+16)
	payload[0] = 0 // version
	binary.BigEndian.PutUint32(payload[12:16], timescale)
	binary.BigEndian.PutUint32(payload[16:20], duration)
	return payload
}

func buildMvhdVersion1Payload(timescale uint32, duration uint64) []byte {
	payload := make([]byte, 4+28)
	payload[0] = 1 // version
	binary.BigEndian.PutUint32(payload[20:24], timescale)
	binary.BigEndian.PutUint64(payload[24:32], duration)
	return payload
}
