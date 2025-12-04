package middleware

import (
	"constructor-script-backend/internal/config"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimitMiddleware creates a middleware that limits request rate per IP
// It requires a RateLimitManager to be set in the context by the application
func RateLimitMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if shouldBypassRateLimit(c.Request) {
			c.Next()
			return
		}

		// Get manager from context
		managerVal, exists := c.Get("rateLimitManager")
		if !exists {
			// If no manager is set, skip rate limiting
			c.Next()
			return
		}

		manager, ok := managerVal.(*RateLimitManager)
		if !ok || manager == nil {
			c.Next()
			return
		}

		limiter := manager.GetVisitor(
			c.ClientIP(),
			cfg.RateLimitRequests,
			cfg.RateLimitWindow,
			cfg.RateLimitBurst,
		)

		if limiter == nil {
			c.Next()
			return
		}

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func shouldBypassRateLimit(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}

	path := r.URL.Path
	if path == "" {
		return false
	}

	staticPrefixes := []string{
		"/static/",
		"/uploads/",
	}

	for _, prefix := range staticPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	switch path {
	case "/favicon.ico":
		return true
	}

	return false
}
