package handlers

import (
	"constructor-script-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func (h *TemplateHandler) currentUser(c *gin.Context) (*models.User, bool) {
	if h.authService == nil {
		return nil, false
	}

	clearCookie := func() {
		secure := c.Request.TLS != nil
		c.SetCookie(authTokenCookieName, "", -1, "/", "", secure, false)
	}

	tokenString, err := c.Cookie(authTokenCookieName)
	if err != nil || tokenString == "" {
		return nil, false
	}

	parsed, err := h.authService.ValidateToken(tokenString)
	if err != nil || parsed == nil || !parsed.Valid {
		clearCookie()
		return nil, false
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		clearCookie()
		return nil, false
	}

	userIDValue, ok := claims["user_id"]
	if !ok {
		clearCookie()
		return nil, false
	}

	var userID uint
	switch value := userIDValue.(type) {
	case float64:
		userID = uint(value)
	case int:
		userID = uint(value)
	default:
		clearCookie()
		return nil, false
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		clearCookie()
		return nil, false
	}

	return user, true
}

func (h *TemplateHandler) addUserContext(c *gin.Context, data gin.H) {
	user, ok := h.currentUser(c)
	if !ok {
		data["IsAuthenticated"] = false
		return
	}

	data["IsAuthenticated"] = true
	data["CurrentUser"] = user
}
