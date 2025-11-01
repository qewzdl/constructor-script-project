package lang

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Default represents the fallback language code used when no explicit language
// is configured. The value follows BCP 47 conventions.
const Default = "en"

var errEmptyCode = errors.New("language code cannot be empty")

// Normalize validates the provided language code and returns it in a
// canonicalised form (lowercase language, uppercase region). Supported formats
// follow the common `ll` or `ll-RR` pattern where `l` is an alphabetic
// character and `R` is the region designator.
func Normalize(code string) (string, error) {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return "", errEmptyCode
	}

	parts := strings.Split(trimmed, "-")
	if len(parts) > 2 {
		return "", fmt.Errorf("invalid language code %q", code)
	}

	language := strings.ToLower(parts[0])
	if len(language) < 2 || len(language) > 8 {
		return "", fmt.Errorf("invalid language code %q", code)
	}
	for _, r := range language {
		if !unicode.IsLetter(r) {
			return "", fmt.Errorf("invalid language code %q", code)
		}
	}

	if len(parts) == 1 {
		return language, nil
	}

	region := parts[1]
	if len(region) < 2 || len(region) > 3 {
		return "", fmt.Errorf("invalid language region in %q", code)
	}
	for _, r := range region {
		if !unicode.IsLetter(r) {
			return "", fmt.Errorf("invalid language region in %q", code)
		}
	}

	region = strings.ToUpper(region)
	return language + "-" + region, nil
}

// NormalizeList normalises a slice of language codes, removing duplicates while
// preserving the order of first occurrence. Empty entries are ignored. If any
// code fails validation the returned slice will be empty alongside the error.
func NormalizeList(codes []string) ([]string, error) {
	if len(codes) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(codes))
	result := make([]string, 0, len(codes))

	for _, raw := range codes {
		if strings.TrimSpace(raw) == "" {
			continue
		}

		normalized, err := Normalize(raw)
		if err != nil {
			return nil, err
		}

		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	return result, nil
}

// EnsureDefault normalises the provided default language and list of supported
// languages, guaranteeing the default is the first entry in the returned list.
func EnsureDefault(defaultCode string, supported []string) (string, []string, error) {
	normalizedDefault, err := Normalize(defaultCode)
	if err != nil {
		return "", nil, fmt.Errorf("invalid default language %q: %w", defaultCode, err)
	}

	normalizedSupported, err := NormalizeList(supported)
	if err != nil {
		return "", nil, err
	}

	result := make([]string, 0, len(normalizedSupported)+1)
	result = append(result, normalizedDefault)
	seen := map[string]struct{}{normalizedDefault: {}}

	for _, code := range normalizedSupported {
		if _, exists := seen[code]; exists {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}

	return normalizedDefault, result, nil
}

// EncodeList serialises the provided list of language codes into a JSON array.
func EncodeList(codes []string) (string, error) {
	data, err := json.Marshal(codes)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DecodeList attempts to parse the provided value as a JSON array of language
// codes. If JSON decoding fails the value is treated as a comma separated list
// instead. The returned slice is not normalised.
func DecodeList(value string) ([]string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	var codes []string
	if err := json.Unmarshal([]byte(trimmed), &codes); err == nil {
		return codes, nil
	}

	parts := strings.Split(trimmed, ",")
	codes = make([]string, 0, len(parts))
	for _, part := range parts {
		if token := strings.TrimSpace(part); token != "" {
			codes = append(codes, token)
		}
	}
	if len(codes) == 0 {
		return nil, errors.New("no language codes found")
	}
	return codes, nil
}
