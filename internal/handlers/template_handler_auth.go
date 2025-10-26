package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"constructor-script-backend/internal/authorization"
	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
)

func (h *TemplateHandler) currentUser(c *gin.Context) (*models.User, bool) {
	if h.authService == nil {
		return nil, false
	}

	clearCookies := func() {
		secure := c.Request.TLS != nil
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie(constants.AuthTokenCookieName, "", -1, "/", "", secure, true)
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie(constants.CSRFTokenCookieName, "", -1, "/", "", secure, false)
	}

	tokenString, err := c.Cookie(constants.AuthTokenCookieName)
	if err != nil || tokenString == "" {
		return nil, false
	}

	parsed, err := h.authService.ValidateToken(tokenString)
	if err != nil || parsed == nil || !parsed.Valid {
		clearCookies()
		return nil, false
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		clearCookies()
		return nil, false
	}

	userIDValue, ok := claims["user_id"]
	if !ok {
		clearCookies()
		return nil, false
	}

	var userID uint
	switch value := userIDValue.(type) {
	case float64:
		userID = uint(value)
	case int:
		userID = uint(value)
	default:
		clearCookies()
		return nil, false
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		clearCookies()
		return nil, false
	}

	return user, true
}

func (h *TemplateHandler) addUserContext(c *gin.Context, data gin.H) {
	data["IsAuthenticated"] = false
	data["IsAdmin"] = false

	user, ok := h.currentUser(c)
	if !ok {
		return
	}

	data["IsAuthenticated"] = true
	data["IsAdmin"] = user.Role == authorization.RoleAdmin
	data["CurrentUser"] = user
	data["UserPermissions"] = gin.H{
		"manage_all_content":  authorization.RoleHasPermission(user.Role, authorization.PermissionManageAllContent),
		"manage_own_content":  authorization.RoleHasPermission(user.Role, authorization.PermissionManageOwnContent),
		"publish_content":     authorization.RoleHasPermission(user.Role, authorization.PermissionPublishContent),
		"moderate_comments":   authorization.RoleHasPermission(user.Role, authorization.PermissionModerateComments),
		"manage_settings":     authorization.RoleHasPermission(user.Role, authorization.PermissionManageSettings),
		"manage_users":        authorization.RoleHasPermission(user.Role, authorization.PermissionManageUsers),
		"manage_themes":       authorization.RoleHasPermission(user.Role, authorization.PermissionManageThemes),
		"manage_backups":      authorization.RoleHasPermission(user.Role, authorization.PermissionManageBackups),
		"manage_navigation":   authorization.RoleHasPermission(user.Role, authorization.PermissionManageNavigation),
		"manage_integrations": authorization.RoleHasPermission(user.Role, authorization.PermissionManageIntegrations),
	}
}
