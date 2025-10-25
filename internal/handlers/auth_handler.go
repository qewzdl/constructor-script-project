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
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

const (
	authTokenTTLSeconds = 72 * 60 * 60
	csrfTokenBytes      = 32
)

func generateCSRFToken() (string, error) {
	token := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token), nil
}

func (h *AuthHandler) setAuthCookie(c *gin.Context, token string, maxAge int) {
	secure := c.Request.TLS != nil
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(constants.AuthTokenCookieName, token, maxAge, "/", "", secure, true)
}

func (h *AuthHandler) setCSRFCookie(c *gin.Context, token string, maxAge int) {
	secure := c.Request.TLS != nil
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(constants.CSRFTokenCookieName, token, maxAge, "/", "", secure, false)
}

func (h *AuthHandler) clearAuthCookie(c *gin.Context) {
	secure := c.Request.TLS != nil
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(constants.AuthTokenCookieName, "", -1, "/", "", secure, true)
}

func (h *AuthHandler) clearCSRFCookie(c *gin.Context) {
	secure := c.Request.TLS != nil
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(constants.CSRFTokenCookieName, "", -1, "/", "", secure, false)
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
	users, err := h.authService.GetAllUsers()
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
