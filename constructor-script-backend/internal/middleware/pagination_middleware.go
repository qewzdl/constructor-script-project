package middleware

import "github.com/gin-gonic/gin"

func PaginationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		page := c.DefaultQuery("page", "1")
		limit := c.DefaultQuery("limit", "10")

		c.Set("page", page)
		c.Set("limit", limit)
		c.Next()
	}
}
