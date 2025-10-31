package sections

import (
	"strings"

	"constructor-script-backend/internal/models"
)

func sectionContent(elem models.SectionElement) map[string]interface{} {
	if contentMap, ok := elem.Content.(map[string]interface{}); ok {
		return contentMap
	}
	return map[string]interface{}{}
}

func getString(content map[string]interface{}, key string) string {
	if content == nil {
		return ""
	}
	if value, ok := content[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func parseBool(value interface{}, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(v))
		if trimmed == "" {
			return fallback
		}
		switch trimmed {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		default:
			return fallback
		}
	default:
		return fallback
	}
}

func normalizeHeading(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	switch trimmed {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return trimmed
	default:
		return ""
	}
}

func normalizeSearchType(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	switch trimmed {
	case "title", "content", "tag", "author", "all":
		return trimmed
	default:
		return ""
	}
}
