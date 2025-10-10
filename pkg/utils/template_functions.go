package utils

import (
	"html/template"
	"strings"
	"time"
)

// GetTemplateFuncs возвращает набор функций для использования в шаблонах
func GetTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		// Математические операции
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},

		// Сравнения
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"le": func(a, b int) bool {
			return a <= b
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"ge": func(a, b int) bool {
			return a >= b
		},

		// Работа со строками
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
			// Простое удаление HTML тегов (для production лучше использовать библиотеку)
			return strings.ReplaceAll(strings.ReplaceAll(s, "<", ""), ">", "")
		},

		// Работа с датами
		"formatDate": func(t time.Time, format string) string {
			layouts := map[string]string{
				"short":    "02.01.2006",
				"medium":   "02 January 2006",
				"long":     "Monday, 02 January 2006",
				"time":     "15:04",
				"datetime": "02.01.2006 15:04",
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
					return "только что"
				}
				return formatPlural(minutes, "минуту", "минуты", "минут") + " назад"
			}

			if duration.Hours() < 24 {
				hours := int(duration.Hours())
				return formatPlural(hours, "час", "часа", "часов") + " назад"
			}

			days := int(duration.Hours() / 24)
			if days < 30 {
				return formatPlural(days, "день", "дня", "дней") + " назад"
			}

			months := days / 30
			if months < 12 {
				return formatPlural(months, "месяц", "месяца", "месяцев") + " назад"
			}

			years := months / 12
			return formatPlural(years, "год", "года", "лет") + " назад"
		},

		// Работа со срезами
		"slice": func(items interface{}, start, end int) interface{} {
			// Упрощенная реализация
			return items
		},
		"first": func(items interface{}, count int) interface{} {
			return items
		},
		"last": func(items interface{}, count int) interface{} {
			return items
		},

		// Условия
		"default": func(value, defaultValue interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},

		// HTML и безопасность
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
		"safeJS": func(s string) template.JS {
			return template.JS(s)
		},

		// Утилиты
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
	}
}

// formatPlural - вспомогательная функция для правильных окончаний
func formatPlural(n int, one, few, many string) string {
	mod10 := n % 10
	mod100 := n % 100

	if mod10 == 1 && mod100 != 11 {
		return formatNumber(n, one)
	}
	if mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20) {
		return formatNumber(n, few)
	}
	return formatNumber(n, many)
}

func formatNumber(n int, word string) string {
	return strings.TrimSpace(strings.Join([]string{intToString(n), word}, " "))
}

func intToString(n int) string {
	return string(rune('0' + n))
}
