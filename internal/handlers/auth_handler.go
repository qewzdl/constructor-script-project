package handlers

import (
	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	courseservice "constructor-script-backend/plugins/courses/service"
)

type AuthHandler struct {
	authService      *service.AuthService
	coursePackageSvc *courseservice.PackageService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) SetCoursePackageService(courseService *courseservice.PackageService) {
	if h == nil {
		return
	}
	h.coursePackageSvc = courseService
}

const (
	authTokenTTLSeconds = 72 * 60 * 60
	csrfTokenBytes      = 32
)

// cookieConfig holds cookie configuration
type cookieConfig struct {
	name     string
	value    string
	maxAge   int
	httpOnly bool
}

func generateCSRFToken() (string, error) {
	token := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token), nil
}

// setCookie is a unified method for setting cookies with proper security settings
func (h *AuthHandler) setCookie(c *gin.Context, cfg cookieConfig) {
	secure := c.Request.TLS != nil
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(cfg.name, cfg.value, cfg.maxAge, "/", "", secure, cfg.httpOnly)
}

func (h *AuthHandler) setAuthCookie(c *gin.Context, token string, maxAge int) {
	h.setCookie(c, cookieConfig{
		name:     constants.AuthTokenCookieName,
		value:    token,
		maxAge:   maxAge,
		httpOnly: true,
	})
}

func (h *AuthHandler) setCSRFCookie(c *gin.Context, token string, maxAge int) {
	h.setCookie(c, cookieConfig{
		name:     constants.CSRFTokenCookieName,
		value:    token,
		maxAge:   maxAge,
		httpOnly: false,
	})
}

func (h *AuthHandler) clearAuthCookie(c *gin.Context) {
	h.setCookie(c, cookieConfig{
		name:     constants.AuthTokenCookieName,
		value:    "",
		maxAge:   -1,
		httpOnly: true,
	})
}

func (h *AuthHandler) clearCSRFCookie(c *gin.Context) {
	h.setCookie(c, cookieConfig{
		name:     constants.CSRFTokenCookieName,
		value:    "",
		maxAge:   -1,
		httpOnly: false,
	})
}

func bindAuthRequest(c *gin.Context, req interface{}) error {
	if strings.Contains(c.GetHeader("Content-Type"), "application/json") {
		return c.ShouldBindJSON(req)
	}
	return c.ShouldBind(req)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := bindAuthRequest(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.Register(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := bindAuthRequest(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, user, err := h.authService.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	csrfToken, err := generateCSRFToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate CSRF token"})
		return
	}

	h.setAuthCookie(c, token, authTokenTTLSeconds)
	h.setCSRFCookie(c, csrfToken, authTokenTTLSeconds)

	c.JSON(http.StatusOK, models.AuthResponse{
		Token:     token,
		User:      *user,
		CSRFToken: csrfToken,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	h.clearAuthCookie(c)
	h.clearCSRFCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *AuthHandler) GetAllUsers(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	limit := 0
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	users, err := h.authService.GetAllUsers(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *AuthHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if err := h.authService.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

func (h *AuthHandler) UpdateUserRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authService.UpdateUserRole(uint(id), req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user role updated successfully"})
}
