package service

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	initialAvatarPrefix = "avatar-initial-"
	initialAvatarExt    = ".png"
)

// EnsureInitialAvatar returns a reusable, on-disk avatar image for the provided username's initial.
// It creates the image once per initial and reuses existing files on subsequent calls.
func (s *UploadService) EnsureInitialAvatar(username string) (string, error) {
	if s == nil {
		return "", errUploadServiceMissing
	}

	glyph, key := resolveInitial(username)
	if glyph == "" || key == "" {
		return "", nil
	}

	filename := fmt.Sprintf("%s%s%s", initialAvatarPrefix, key, initialAvatarExt)
	filePath := filepath.Join(s.uploadDir, filename)
	url := "/uploads/" + filename

	if _, err := os.Stat(filePath); err == nil {
		return url, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	img, err := renderInitialAvatarImage(glyph)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}

	tmp, err := os.CreateTemp(s.uploadDir, filename+".tmp-*")
	if err != nil {
		return "", err
	}

	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}

	if err := os.Rename(tmp.Name(), filePath); err != nil {
		if errors.Is(err, os.ErrExist) {
			os.Remove(tmp.Name())
			return url, nil
		}
		os.Remove(tmp.Name())
		return "", err
	}

	return url, nil
}

// IsInitialAvatar reports whether the provided URL points to a generated initial avatar.
func (s *UploadService) IsInitialAvatar(url string) bool {
	if url == "" {
		return false
	}

	filename := filepath.Base(url)
	return strings.HasPrefix(filename, initialAvatarPrefix)
}

func resolveInitial(username string) (string, string) {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return "", ""
	}

	r, _ := utf8.DecodeRuneInString(trimmed)
	if r == utf8.RuneError {
		return "", ""
	}

	glyph := strings.ToUpper(string(r))
	key := strings.ToLower(glyph)

	if len(key) != 1 || !isASCIIAlphaNumeric(key[0]) {
		key = fmt.Sprintf("u%x", r)
	}

	return glyph, key
}

func isASCIIAlphaNumeric(value byte) bool {
	return (value >= 'a' && value <= 'z') || (value >= '0' && value <= '9')
}

func renderInitialAvatarImage(letter string) (image.Image, error) {
	const size = 256

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	background := color.RGBA{R: 17, G: 24, B: 39, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{background}, image.Point{}, draw.Src)

	face, err := loadMonoFace(float64(size) * 0.5)
	if err != nil {
		return nil, err
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.White),
		Face: face,
	}

	bounds, _ := font.BoundString(face, letter)
	textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
	textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

	x := (size - textWidth) / 2
	verticalAdjust := int(math.Round(float64(size) * 0.05))
	y := (size+textHeight)/2 - verticalAdjust

	d.Dot = fixed.P(x, y)
	d.DrawString(letter)

	return img, nil
}

func loadMonoFace(size float64) (font.Face, error) {
	fontData, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return nil, err
	}

	return opentype.NewFace(fontData, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingNone,
	})
}
