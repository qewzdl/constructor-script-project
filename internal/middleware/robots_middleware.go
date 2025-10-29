package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const defaultRobotsDirectives = "noindex, nofollow"

// NoIndexMiddleware sets the X-Robots-Tag header with the provided directives or a
// sensible default to prevent the response from being indexed by search engines.
func NoIndexMiddleware(directives ...string) gin.HandlerFunc {
	value := defaultRobotsDirectives

	if len(directives) > 0 {
		cleaned := make([]string, 0, len(directives))
		for _, directive := range directives {
			directive = strings.TrimSpace(directive)
			if directive == "" {
				continue
			}
			cleaned = append(cleaned, directive)
		}

		if len(cleaned) > 0 {
			value = strings.Join(cleaned, ", ")
		}
	}

	return func(c *gin.Context) {
		c.Header("X-Robots-Tag", value)
		c.Next()
	}
}
