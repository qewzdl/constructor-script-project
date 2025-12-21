package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"constructor-script-backend/internal/constants"

	"github.com/gin-gonic/gin"
)

var stateChangingMethods = map[string]struct{}{
	http.MethodPost:   {},
	http.MethodPut:    {},
	http.MethodPatch:  {},
	http.MethodDelete: {},
}

var csrfExemptPaths = map[string]struct{}{
	"/api/v1/setup":                   {},
	"/api/v1/logout":                  {},
	"/api/v1/courses/checkout/verify": {},
}

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, shouldCheck := stateChangingMethods[c.Request.Method]; !shouldCheck {
			c.Next()
			return
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		if _, exempt := csrfExemptPaths[path]; exempt {
			c.Next()
			return
		}

		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader != "" {
			c.Next()
			return
		}

		tokenCookie, err := c.Cookie(constants.AuthTokenCookieName)
		if err != nil || strings.TrimSpace(tokenCookie) == "" {
			c.Next()
			return
		}

		csrfCookie, err := c.Cookie(constants.CSRFTokenCookieName)
		if err != nil || strings.TrimSpace(csrfCookie) == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "missing CSRF token"})
			return
		}

		headerToken := strings.TrimSpace(c.GetHeader("X-CSRF-Token"))
		if headerToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "missing CSRF header"})
			return
		}

		if subtle.ConstantTimeCompare([]byte(csrfCookie), []byte(headerToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid CSRF token"})
			return
		}

		c.Next()
	}
}
