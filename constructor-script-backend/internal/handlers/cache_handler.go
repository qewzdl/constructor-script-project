package handlers

import (
	"constructor-script-backend/pkg/cache"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ClearCache(cacheService *cache.Cache) gin.HandlerFunc {
	return func(c *gin.Context) {
		cacheType := c.DefaultQuery("type", "all")

		var err error

		switch cacheType {
		case "posts":
			err = cacheService.InvalidatePostsCache()
		case "categories":
			err = cacheService.DeletePattern("category:*")
		case "all":
			err = cacheService.FlushAll()
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cache type"})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "cache cleared successfully",
			"type":    cacheType,
		})
	}
}
