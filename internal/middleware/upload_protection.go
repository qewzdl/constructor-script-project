package middleware

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

var publicUploadExtensions = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".gif":  {},
	".webp": {},
	".ico":  {},
	".svg":  {},
}

// UploadsProtection blocks direct access to non-public upload types (videos, documents, subtitles)
// through the public /uploads route, forcing protected asset handlers to be used instead.
func UploadsProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		rawPath := strings.ToLower(strings.TrimSpace(c.Param("filepath")))
		ext := strings.ToLower(filepath.Ext(rawPath))
		if ext != "" {
			if _, ok := publicUploadExtensions[ext]; ok {
				c.Next()
				return
			}
		}

		c.AbortWithStatus(http.StatusNotFound)
	}
}
