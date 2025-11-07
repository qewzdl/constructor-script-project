package handlers

import (
        "net/http"
        "strconv"

        "github.com/gin-gonic/gin"
)

func parseUintParam(c *gin.Context, name string) (uint, bool) {
        value, err := strconv.ParseUint(c.Param(name), 10, 64)
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid identifier"})
                return 0, false
        }
        return uint(value), true
}
