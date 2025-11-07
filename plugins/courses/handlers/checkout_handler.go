package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	courseservice "constructor-script-backend/plugins/courses/service"
)

// CheckoutHandler exposes course checkout operations to HTTP clients.
type CheckoutHandler struct {
	service *courseservice.CheckoutService
}

// NewCheckoutHandler constructs a handler instance.
func NewCheckoutHandler(service *courseservice.CheckoutService) *CheckoutHandler {
	return &CheckoutHandler{service: service}
}

// SetService updates the checkout service dependency.
func (h *CheckoutHandler) SetService(service *courseservice.CheckoutService) {
	if h == nil {
		return
	}
	h.service = service
}

func (h *CheckoutHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "course checkout service unavailable"})
		return false
	}
	return true
}

// CreateSession starts a new checkout session for the requested course package.
func (h *CheckoutHandler) CreateSession(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	var req models.CourseCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.service.CreateCheckoutSession(c.Request.Context(), req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.CourseCheckoutSession{
		SessionID:   session.ID,
		CheckoutURL: session.URL,
	})
}

func (h *CheckoutHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, courseservice.ErrCheckoutDisabled):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "course checkout disabled"})
	case errors.Is(err, courseservice.ErrInvalidPackagePrice):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "course package not found"})
	default:
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
	}
}
