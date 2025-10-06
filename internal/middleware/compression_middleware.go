package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func CompressionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		c.Header("Content-Encoding", "gzip")
		c.Next()
	}
}
