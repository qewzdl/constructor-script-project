package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

var (
	setupStatusCache struct {
		sync.RWMutex
		complete  bool
		timestamp time.Time
		ttl       time.Duration
	}
)

func init() {
	setupStatusCache.ttl = 30 * time.Second
}

func SetupMiddleware(setupService *service.SetupService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if setupService == nil {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		method := c.Request.Method

		// Always allow OPTIONS requests for CORS
		if method == http.MethodOptions {
			c.Next()
			return
		}

		// Check if accessing setup endpoints
		if strings.HasPrefix(path, "/setup") || strings.HasPrefix(path, "/api/v1/setup") {
			// If SETUP_KEY is configured, verify it
			if cfg.SetupKey != "" {
				providedKey := c.Query("key")
				if providedKey == "" {
					providedKey = c.GetHeader("X-Setup-Key")
				}

				if providedKey != cfg.SetupKey {
					// Log the access attempt
					logger.Warn("Setup access denied - invalid or missing key", map[string]interface{}{
						"path":   path,
						"method": method,
						"hasKey": providedKey != "",
					})

					// For GET request to /setup page (not API), redirect to key entry page
					if method == http.MethodGet && path == "/setup" {
						c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
						c.Redirect(http.StatusTemporaryRedirect, "/setup/key-required")
						c.Abort()
						return
					}

					// For all other requests (including API), return JSON error
					c.Header("Content-Type", "application/json; charset=utf-8")
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"error": "Invalid or missing setup key",
					})
					return
				}

				// Valid key - log successful access
				logger.Info("Setup access granted with valid key", map[string]interface{}{
					"path": path,
				})
			} else {
				// SETUP_KEY not configured - allow access but log warning
				logger.Warn("Setup access without key verification - SETUP_KEY not configured", map[string]interface{}{
					"path": path,
				})
			}
			// Key is valid or not required, allow access to setup
			c.Next()
			return
		}

		// Allow certain paths during setup
		if allowDuringSetup(path) {
			c.Next()
			return
		}

		// Check setup status with caching
		complete, err := getCachedSetupStatus(setupService)
		if err != nil {
			logger.Error(err, "Failed to determine setup status", map[string]interface{}{
				"path":   path,
				"method": method,
			})
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to verify setup status",
			})
			return
		}

		// If setup is complete, allow all requests
		if complete {
			c.Next()
			return
		}

		// Setup not complete - redirect or return error
		if method == http.MethodGet {
			// For GET requests, redirect to setup page
			c.Redirect(http.StatusTemporaryRedirect, "/setup")
			c.Abort()
			return
		}

		// For other methods, return JSON error
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"error":          "Setup required",
			"setup_required": true,
		})
	}
}

// getCachedSetupStatus checks setup status with caching to reduce database queries
func getCachedSetupStatus(setupService *service.SetupService) (bool, error) {
	setupStatusCache.RLock()
	if time.Since(setupStatusCache.timestamp) < setupStatusCache.ttl {
		complete := setupStatusCache.complete
		setupStatusCache.RUnlock()
		return complete, nil
	}
	setupStatusCache.RUnlock()

	// Cache expired, fetch new status
	complete, err := setupService.IsSetupComplete()
	if err != nil {
		return false, err
	}

	// Update cache
	setupStatusCache.Lock()
	setupStatusCache.complete = complete
	setupStatusCache.timestamp = time.Now()
	setupStatusCache.Unlock()

	return complete, nil
}

// InvalidateSetupCache clears the setup status cache
func InvalidateSetupCache() {
	setupStatusCache.Lock()
	setupStatusCache.timestamp = time.Time{}
	setupStatusCache.Unlock()
}

func allowDuringSetup(path string) bool {
	allowedPrefixes := []string{
		"/setup",
		"/api/v1/setup",
		"/static/",
		"/uploads/",
	}

	allowedExact := map[string]struct{}{
		"/health":      {},
		"/metrics":     {},
		"/favicon.ico": {},
		"/robots.txt":  {},
	}

	// Check exact matches first
	if _, ok := allowedExact[path]; ok {
		return true
	}

	// Check prefix matches
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}
