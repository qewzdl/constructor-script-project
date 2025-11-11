package validator

import (
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
	if len(password) < 8 {
		return false, "password must be at least 8 characters long"
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasUpper || !hasLower || !hasNumber {
		return false, "password must contain uppercase, lowercase and numbers"
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
