package middleware

import (
	"net/http"
	"strings"

	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

func SetupMiddleware(setupService service.SetupUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		if setupService == nil {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		method := c.Request.Method

		if method == http.MethodOptions {
			c.Next()
			return
		}

		if allowDuringSetup(path) {
			c.Next()
			return
		}

		complete, err := setupService.IsSetupComplete()
		if err != nil {
			logger.Error(err, "Failed to determine setup status", map[string]interface{}{"path": path})
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify setup status"})
			return
		}

		if complete {
			c.Next()
			return
		}

		if method == http.MethodGet {
			c.Redirect(http.StatusTemporaryRedirect, "/setup")
			c.Abort()
			return
		}

		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Setup required"})
	}
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
	}

	if _, ok := allowedExact[path]; ok {
		return true
	}

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}
