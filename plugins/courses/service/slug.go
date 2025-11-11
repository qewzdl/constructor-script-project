package service

import "strings"

func normalizeSlug(value string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return ""
	}
	return strings.ToLower(cleaned)
}
