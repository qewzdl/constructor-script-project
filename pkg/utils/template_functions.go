package utils

import (
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type AssetModTimeFunc func(path string) (time.Time, error)

func GetTemplateFuncs(assetModTime AssetModTimeFunc) template.FuncMap {
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

		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"title":     strings.Title,
		"trim":      strings.TrimSpace,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"contains":  strings.Contains,
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"stripHTML": func(s string) string {
			return strings.ReplaceAll(strings.ReplaceAll(s, "<", ""), ">", "")
		},
		"pathEquals": func(current, value string) bool {
			current = NormalizePath(current)
			value = strings.TrimSpace(value)
			if value == "" {
				return false
			}
			if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
				if parsed, err := url.Parse(value); err == nil {
					if parsed.Path != "" {
						value = parsed.Path
					}
				}
			}
			value = NormalizePath(value)
			return current != "" && current == value
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

		"default": func(defaultValue, value interface{}) interface{} {
			if isEmpty(value) {
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
			version := int64(0)
			if assetModTime != nil {
				if modTime, err := assetModTime(path); err == nil {
					version = modTime.Unix()
				}
			}
			if version == 0 {
				trimmed := strings.TrimPrefix(path, "/")
				fsPath := filepath.FromSlash(trimmed)
				if info, err := os.Stat(fsPath); err == nil {
					version = info.ModTime().Unix()
				}
			}
			if version == 0 {
				return path
			}
			separator := "?"
			if strings.Contains(path, "?") {
				separator = "&"
			}
			return fmt.Sprintf("%s%sv=%d", path, separator, version)
		},

		"formatBytes": func(n int64) string {
			if n <= 0 {
				return "0 B"
			}
			f := float64(n)
			units := []string{"B", "KB", "MB", "GB", "TB"}
			i := 0
			for f >= 1024 && i < len(units)-1 {
				f = f / 1024
				i++
			}
			if units[i] == "B" {
				return fmt.Sprintf("%d %s", int64(f), units[i])
			}
			return fmt.Sprintf("%.2f %s", f, units[i])
		},

		"guessFileType": func(fileType, mimeType, urlStr string) string {
			ft := strings.TrimSpace(strings.ToLower(fileType))
			mt := strings.TrimSpace(strings.ToLower(mimeType))
			if ft != "" {
				return strings.Title(ft)
			}
			if mt != "" {
				switch {
				case strings.HasPrefix(mt, "image/"):
					return "Image"
				case strings.HasPrefix(mt, "video/"):
					return "Video"
				case strings.HasPrefix(mt, "audio/"):
					return "Audio"
				case mt == "application/pdf", strings.HasPrefix(mt, "text/"):
					return "Document"
				case strings.Contains(mt, "zip") || strings.Contains(mt, "compressed") || strings.Contains(mt, "gzip") || strings.Contains(mt, "tar"):
					return "Archive"
				}
				return mt
			}
			lower := strings.ToLower(strings.TrimSpace(urlStr))
			ext := strings.ToLower(filepath.Ext(lower))
			switch ext {
			case ".pdf", ".doc", ".docx", ".txt", ".rtf", ".odt", ".xls", ".xlsx", ".ppt", ".pptx":
				return "Document"
			case ".zip", ".tar", ".gz", ".7z", ".rar", ".bz2":
				return "Archive"
			case ".jpg", ".jpeg", ".png", ".gif", ".svg", ".bmp", ".webp", ".ico":
				return "Image"
			case ".mp4", ".mov", ".webm", ".avi", ".mkv", ".flv":
				return "Video"
			case ".mp3", ".wav", ".ogg", ".flac", ".aac", ".m4a":
				return "Audio"
			}
			return "Not specified"
		},
	}
}

func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.String:
		return strings.TrimSpace(v.String()) == ""
	case reflect.Bool:
		return false
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Slice, reflect.Array, reflect.Map:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	}

	zero := reflect.Zero(v.Type())
	return reflect.DeepEqual(value, zero.Interface())
}

func formatPlural(n int, one, few, many string) string {
	if n == 1 {
		return formatNumber(n, one)
	}
	return formatNumber(n, many)
}

func formatNumber(n int, word string) string {
	return strings.TrimSpace(strings.Join([]string{intToString(n), word}, " "))
}

func intToString(n int) string {
	return fmt.Sprintf("%d", n)
}

func NormalizePath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "/"
	}

	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		if parsed, err := url.Parse(trimmed); err == nil {
			if parsed.Path != "" {
				trimmed = parsed.Path
			} else {
				trimmed = "/"
			}
		}
	}

	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}

	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == "" {
		return "/"
	}

	if cleaned != "/" && strings.HasSuffix(cleaned, "/") {
		cleaned = strings.TrimSuffix(cleaned, "/")
	}

	return cleaned
}
