package validator

import (
	"bytes"
	"mime"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/microcosm-cc/bluemonday"
)

var (
	validate  *validator.Validate
	sanitizer *bluemonday.Policy
)

func Init() {
	validate = validator.New()

	sanitizer = bluemonday.UGCPolicy()

	registerCustomValidations(validate)

	if engine, ok := binding.Validator.Engine().(*validator.Validate); ok {
		registerCustomValidations(engine)
	}
}

func registerCustomValidations(v *validator.Validate) {
	v.RegisterValidation("username", validateUsername)
	v.RegisterValidation("slug", validateSlug)
	v.RegisterValidation("no_html", validateNoHTML)
}

func Validate(s interface{}) error {
	return validate.Struct(s)
}

func SanitizeHTML(html string) string {
	return sanitizer.Sanitize(html)
}

func SanitizeString(s string) string {
	return bluemonday.StrictPolicy().Sanitize(s)
}

func ValidateEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func ValidatePassword(password string) (bool, string) {
	if len(password) < 6 {
		return false, "password must be at least 6 characters long"
	}

	return true, ""
}

func validateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, username)
	return matched && len(username) >= 3 && len(username) <= 30
}

func validateSlug(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	matched, _ := regexp.MatchString(`^[a-z0-9-]+$`, slug)
	return matched
}

func validateNoHTML(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return !strings.Contains(value, "<") && !strings.Contains(value, ">")
}

func TrimSpaces(s string) string {
	return strings.TrimSpace(s)
}

func NormalizeSpaces(s string) string {
	space := regexp.MustCompile(`\s+`)
	return space.ReplaceAllString(s, " ")
}

func ValidateURL(url string) bool {
	urlRegex := regexp.MustCompile(`^https?://[a-zA-Z0-9\-\.]+\.[a-zA-Z]{2,}(/.*)?$`)
	return urlRegex.MatchString(url)
}

func SanitizeFilename(filename string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	return reg.ReplaceAllString(filename, "_")
}

func ValidateImageExtension(filename string) bool {
	allowedExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".ico"}
	filename = strings.ToLower(filename)

	for _, ext := range allowedExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}

func ValidateFileSize(size int64, maxSize int64) bool {
	return size > 0 && size <= maxSize
}

func EscapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func ValidateIPAddress(ip string) bool {
	ipRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	return ipRegex.MatchString(ip)
}

// MIME Type Validation

// ValidateContentType validates that the provided MIME type is in the allowed list
func ValidateContentType(contentType string, allowedMimeTypes []string) bool {
	if contentType == "" || len(allowedMimeTypes) == 0 {
		return false
	}

	// Parse content type and extract the base type (e.g., "image/png" from "image/png; charset=utf-8")
	mimeType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	mimeType = strings.ToLower(strings.TrimSpace(mimeType))

	// Check exact matches and wildcard patterns
	for _, allowed := range allowedMimeTypes {
		allowed = strings.ToLower(strings.TrimSpace(allowed))

		// Exact match
		if mimeType == allowed {
			return true
		}

		// Wildcard match (e.g., "image/*" matches "image/png")
		if strings.HasSuffix(allowed, "/*") {
			prefix := strings.TrimSuffix(allowed, "/*")
			if strings.HasPrefix(mimeType, prefix+"/") {
				return true
			}
		}
	}

	return false
}

// DetectFileType attempts to detect the actual MIME type from file content
// Returns the detected MIME type or empty string if detection fails
func DetectFileType(data []byte) string {
	// Check magic numbers for common file types
	if len(data) == 0 {
		return ""
	}

	// Image formats
	if bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47}) {
		return "image/png"
	}
	if bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) {
		return "image/jpeg"
	}
	if bytes.HasPrefix(data, []byte{0x47, 0x49, 0x46, 0x38}) {
		return "image/gif"
	}
	if bytes.HasPrefix(data, []byte{0x52, 0x49, 0x46, 0x46}) && len(data) > 12 &&
		bytes.HasPrefix(data[8:], []byte{0x57, 0x45, 0x42, 0x50}) {
		return "image/webp"
	}
	if bytes.HasPrefix(data, []byte{0x3C, 0x3F, 0x78, 0x6D, 0x6C}) {
		return "text/xml"
	}
	if bytes.HasPrefix(data, []byte{0x3C, 0x21, 0x44, 0x4F, 0x43, 0x54, 0x59, 0x50, 0x45}) {
		return "text/html"
	}

	// PDF
	if bytes.HasPrefix(data, []byte{0x25, 0x50, 0x44, 0x46}) {
		return "application/pdf"
	}

	// ZIP-based formats
	if bytes.HasPrefix(data, []byte{0x50, 0x4B, 0x03, 0x04}) {
		// Could be ZIP, DOCX, XLSX, etc.
		// For now, return generic application/zip
		return "application/zip"
	}

	// MP4
	if len(data) > 12 && (bytes.HasPrefix(data[4:], []byte{0x66, 0x74, 0x79, 0x70})) {
		return "video/mp4"
	}

	// MOV
	if bytes.HasPrefix(data, []byte{0x00, 0x00, 0x00, 0x14, 0x66, 0x74, 0x79, 0x70}) ||
		bytes.HasPrefix(data, []byte{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70}) {
		return "video/quicktime"
	}

	// VTT subtitles
	if bytes.HasPrefix(data, []byte("WEBVTT")) {
		return "text/vtt"
	}

	// TXT - very basic detection
	if isProbablyText(data) {
		return "text/plain"
	}

	return ""
}

// isProbablyText checks if data looks like text by checking for null bytes
func isProbablyText(data []byte) bool {
	// If less than 512 bytes, check all
	checkSize := 512
	if len(data) < checkSize {
		checkSize = len(data)
	}

	for i := 0; i < checkSize; i++ {
		// Null bytes typically indicate binary data
		if data[i] == 0 {
			return false
		}
	}
	return true
}

// ValidateImageContentType validates image MIME types
func ValidateImageContentType(contentType string) bool {
	allowedMimeTypes := []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
		"image/svg+xml",
		"image/x-icon",
	}
	return ValidateContentType(contentType, allowedMimeTypes)
}

// ValidateVideoContentType validates video MIME types
func ValidateVideoContentType(contentType string) bool {
	allowedMimeTypes := []string{
		"video/mp4",
		"video/quicktime",
		"video/x-m4v",
	}
	return ValidateContentType(contentType, allowedMimeTypes)
}

// ValidateDocumentContentType validates document MIME types
func ValidateDocumentContentType(contentType string) bool {
	allowedMimeTypes := []string{
		"application/pdf",
		"text/plain",
		"text/csv",
		"application/json",
		"text/xml",
		"application/xml",
		"text/markdown",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/zip",
		"application/gzip",
		"application/x-tar",
		"application/x-rar-compressed",
		"application/x-7z-compressed",
		"text/vtt",
	}
	return ValidateContentType(contentType, allowedMimeTypes)
}
