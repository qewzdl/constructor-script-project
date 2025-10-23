package middleware

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"constructor-script-backend/internal/models"
)

type ContentSecurityPolicySource interface {
	ContentSecurityPolicyDirectives() models.ContentSecurityPolicyDirectives
}

var baseContentSecurityPolicy = map[string][]string{
	"default-src":     {"'self'"},
	"object-src":      {"'none'"},
	"base-uri":        {"'self'"},
	"frame-ancestors": {"'none'"},
	"form-action":     {"'self'"},
	"script-src": {
		"'self'",
		"'unsafe-inline'",
		"https://pagead2.googlesyndication.com",
		"https://securepubads.g.doubleclick.net",
		"https://www.googletagservices.com",
		"https://ep2.adtrafficquality.google",
	},
	"style-src": {
		"'self'",
		"'unsafe-inline'",
		"https://fonts.googleapis.com",
	},
	"font-src": {"'self'", "https://fonts.gstatic.com", "data:"},
	"img-src": {
		"'self'",
		"data:",
		"https:",
		"https://pagead2.googlesyndication.com",
		"https://tpc.googlesyndication.com",
	},
	"connect-src": {
		"'self'",
		"https://pagead2.googlesyndication.com",
		"https://googleads.g.doubleclick.net",
		"https://ep2.adtrafficquality.google",
	},
	"frame-src": {
		"https://adservice.google.com",
		"https://googleads.g.doubleclick.net",
		"https://tpc.googlesyndication.com",
		"https://ep2.adtrafficquality.google",
		"https://www.google.com",
	},
}

var cspDirectiveOrder = []string{
	"default-src",
	"object-src",
	"base-uri",
	"frame-ancestors",
	"form-action",
	"script-src",
	"style-src",
	"font-src",
	"img-src",
	"connect-src",
	"child-src",
	"frame-src",
}

func SecurityHeadersMiddleware(sources ...ContentSecurityPolicySource) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("X-DNS-Prefetch-Control", "off")
		c.Header("X-Download-Options", "noopen")
		c.Header("X-Permitted-Cross-Domain-Policies", "none")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")
		c.Header("Cross-Origin-Embedder-Policy", "same-origin")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Header("Content-Security-Policy", buildContentSecurityPolicy(sources))

		c.Next()
	}
}

func buildContentSecurityPolicy(sources []ContentSecurityPolicySource) string {
	directives := make(map[string]map[string]struct{}, len(baseContentSecurityPolicy))
	for directive, values := range baseContentSecurityPolicy {
		directives[directive] = make(map[string]struct{}, len(values))
		for _, value := range values {
			directives[directive][value] = struct{}{}
		}
	}

	for _, source := range sources {
		if source == nil {
			continue
		}

		extras := source.ContentSecurityPolicyDirectives()
		for directive, values := range extras {
			if len(values) == 0 {
				continue
			}

			bucket, ok := directives[directive]
			if !ok {
				bucket = make(map[string]struct{}, len(values))
				directives[directive] = bucket
			}

			for _, value := range values {
				value = strings.TrimSpace(value)
				if value == "" {
					continue
				}
				bucket[value] = struct{}{}
			}
		}
	}

	return serializeContentSecurityPolicy(directives)
}

func serializeContentSecurityPolicy(directives map[string]map[string]struct{}) string {
	if len(directives) == 0 {
		return ""
	}

	var parts []string
	used := make(map[string]struct{}, len(cspDirectiveOrder))

	for _, directive := range cspDirectiveOrder {
		if formatted := formatDirective(directive, directives[directive]); formatted != "" {
			parts = append(parts, formatted)
			used[directive] = struct{}{}
		}
	}

	remaining := make([]string, 0, len(directives))
	for directive := range directives {
		if _, ok := used[directive]; ok {
			continue
		}
		if len(directives[directive]) == 0 {
			continue
		}
		remaining = append(remaining, directive)
	}

	sort.Strings(remaining)
	for _, directive := range remaining {
		parts = append(parts, formatDirective(directive, directives[directive]))
	}

	return strings.Join(parts, "; ")
}

func formatDirective(name string, values map[string]struct{}) string {
	if len(values) == 0 {
		return ""
	}

	ordered := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	if baseValues, ok := baseContentSecurityPolicy[name]; ok {
		for _, value := range baseValues {
			if _, exists := values[value]; exists {
				ordered = append(ordered, value)
				seen[value] = struct{}{}
			}
		}
	}

	extras := make([]string, 0, len(values))
	for value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		extras = append(extras, value)
	}

	sort.Strings(extras)
	ordered = append(ordered, extras...)

	return name + " " + strings.Join(ordered, " ")
}
