package utils

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GetTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},

		"eq": func(a, b interface{}) bool { return a == b },
		"ne": func(a, b interface{}) bool { return a != b },
		"lt": func(a, b int) bool { return a < b },
		"le": func(a, b int) bool { return a <= b },
		"gt": func(a, b int) bool { return a > b },
		"ge": func(a, b int) bool { return a >= b },

		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"trim":  strings.TrimSpace,
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"stripHTML": func(s string) string {
			return strings.ReplaceAll(strings.ReplaceAll(s, "<", ""), ">", "")
		},

		"formatDate": func(t time.Time, format string) string {
			layouts := map[string]string{
				"short":    "01/02/2006",
				"medium":   "January 02, 2006",
				"long":     "Monday, January 02, 2006",
				"time":     "15:04",
				"datetime": "01/02/2006 15:04",
				"iso":      time.RFC3339,
			}
			if layout, ok := layouts[format]; ok {
				return t.Format(layout)
			}
			return t.Format(format)
		},
		"timeAgo": func(t time.Time) string {
			duration := time.Since(t)
			if duration.Hours() < 1 {
				minutes := int(duration.Minutes())
				if minutes < 1 {
					return "just now"
				}
				return formatPlural(minutes, "minute", "minutes", "minutes") + " ago"
			}
			if duration.Hours() < 24 {
				hours := int(duration.Hours())
				return formatPlural(hours, "hour", "hours", "hours") + " ago"
			}
			days := int(duration.Hours() / 24)
			if days < 30 {
				return formatPlural(days, "day", "days", "days") + " ago"
			}
			months := days / 30
			if months < 12 {
				return formatPlural(months, "month", "months", "months") + " ago"
			}
			years := months / 12
			return formatPlural(years, "year", "years", "years") + " ago"
		},

		"slice": func(items interface{}, start, end int) interface{} { return items },
		"first": func(items interface{}, count int) interface{} { return items },
		"last":  func(items interface{}, count int) interface{} { return items },

		"default": func(value, defaultValue interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},

		"safe":    func(s string) template.HTML { return template.HTML(s) },
		"safeURL": func(s string) template.URL { return template.URL(s) },
		"safeJS":  func(s string) template.JS { return template.JS(s) },

		"dict": func(values ...interface{}) map[string]interface{} {
			dict := make(map[string]interface{})
			for i := 0; i < len(values); i += 2 {
				key := values[i].(string)
				dict[key] = values[i+1]
			}
			return dict
		},
		"seq": func(n int) []int {
			result := make([]int, n)
			for i := 0; i < n; i++ {
				result[i] = i + 1
			}
			return result
		},
		"asset": func(path string) string {
			if path == "" {
				return ""
			}
			lowerPath := strings.ToLower(path)
			if strings.HasPrefix(lowerPath, "http://") || strings.HasPrefix(lowerPath, "https://") || strings.HasPrefix(path, "//") {
				return path
			}
			trimmed := strings.TrimPrefix(path, "/")
			fsPath := filepath.FromSlash(trimmed)
			info, err := os.Stat(fsPath)
			if err != nil {
				return path
			}
			version := info.ModTime().Unix()
			separator := "?"
			if strings.Contains(path, "?") {
				separator = "&"
			}
			return fmt.Sprintf("%s%sv=%d", path, separator, version)
		},
	}
}

func formatPlural(n int, one, few, many string) string {
	abs := n
	if abs < 0 {
		abs = -abs
	}

	mod100 := abs % 100
	if mod100 >= 11 && mod100 <= 14 {
		return formatNumber(n, many)
	}

	switch abs % 10 {
	case 1:
		return formatNumber(n, one)
	case 2, 3, 4:
		if few != "" {
			return formatNumber(n, few)
		}
	}

	return formatNumber(n, many)
}

func formatNumber(n int, word string) string {
	return strings.TrimSpace(strings.Join([]string{intToString(n), word}, " "))
}

func intToString(n int) string {
	return fmt.Sprintf("%d", n)
}
