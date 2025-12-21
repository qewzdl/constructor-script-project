package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/payments/stripe"
	"constructor-script-backend/pkg/logger"
	courseservice "constructor-script-backend/plugins/courses/service"
)

// CheckoutHandler exposes course checkout operations to HTTP clients.
type CheckoutHandler struct {
	service        *courseservice.CheckoutService
	packageService *courseservice.PackageService
	webhookSecret  string
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

// SetPackageService updates the course package service dependency.
func (h *CheckoutHandler) SetPackageService(service *courseservice.PackageService) {
	if h == nil {
		return
	}
	h.packageService = service
}

// SetWebhookSecret updates the Stripe webhook signing secret.
func (h *CheckoutHandler) SetWebhookSecret(secret string) {
	if h == nil {
		return
	}
	h.webhookSecret = strings.TrimSpace(secret)
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

	if h.packageService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "course package service unavailable"})
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var req models.CourseCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.UserID = userID
	if req.CustomerEmail == "" {
		if email := strings.TrimSpace(c.GetString("email")); email != "" {
			req.CustomerEmail = email
		}
	}

	if owned, err := h.packageService.GetForUser(req.PackageID, userID); err == nil && owned != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "you already own this course"})
		return
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error(err, "Failed to check existing course access", map[string]interface{}{"package_id": req.PackageID, "user_id": userID})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to start checkout"})
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

type stripeCheckoutSession struct {
	ID            string            `json:"id"`
	Status        string            `json:"status"`
	PaymentStatus string            `json:"payment_status"`
	Metadata      map[string]string `json:"metadata"`
	CustomerEmail string            `json:"customer_email"`
}

type stripeWebhookEvent struct {
	Type string `json:"type"`
	Data struct {
		Object stripeCheckoutSession `json:"object"`
	} `json:"data"`
}

// HandleWebhook processes Stripe checkout webhook events and grants course access.
func (h *CheckoutHandler) HandleWebhook(c *gin.Context) {
	requestID := strings.TrimSpace(c.GetString("request_id"))

	if h == nil || h.packageService == nil {
		logger.Warn("Course webhook unavailable: package service missing", map[string]interface{}{
			"request_id": requestID,
		})
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "course package service unavailable"})
		return
	}

	secret := strings.TrimSpace(h.webhookSecret)
	if secret == "" {
		logger.Warn("Course webhook secret not configured", map[string]interface{}{
			"request_id": requestID,
		})
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "stripe webhook not configured"})
		return
	}

	signature := strings.TrimSpace(c.GetHeader("Stripe-Signature"))
	if signature == "" {
		logger.Warn("Stripe webhook missing signature header", map[string]interface{}{
			"request_id": requestID,
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing stripe signature"})
		return
	}

	payload, err := c.GetRawData()
	if err != nil {
		logger.Error(err, "Failed to read Stripe webhook payload", map[string]interface{}{
			"request_id": requestID,
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read webhook payload"})
		return
	}

	if err := stripe.VerifyWebhookSignature(payload, signature, secret, 5*time.Minute); err != nil {
		logger.Warn("Invalid Stripe webhook signature", map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook signature"})
		return
	}

	var event stripeWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		logger.Warn("Invalid Stripe webhook payload", map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook payload"})
		return
	}

	if !strings.EqualFold(event.Type, "checkout.session.completed") {
		logger.Info("Ignored Stripe webhook event", map[string]interface{}{
			"request_id": requestID,
			"event_type": event.Type,
		})
		c.Status(http.StatusOK)
		return
	}

	session := event.Data.Object
	if strings.ToLower(strings.TrimSpace(session.PaymentStatus)) != "paid" && strings.ToLower(strings.TrimSpace(session.Status)) != "complete" {
		logger.Info("Checkout session not paid yet", map[string]interface{}{
			"request_id":      requestID,
			"event_type":      event.Type,
			"session_id":      session.ID,
			"payment_status":  session.PaymentStatus,
			"session_status":  session.Status,
			"customer_email":  session.CustomerEmail,
		})
		c.Status(http.StatusOK)
		return
	}

	metadata := session.Metadata
	if len(metadata) == 0 {
		logger.Warn("Stripe checkout session missing metadata", map[string]interface{}{
			"request_id": requestID,
			"event_type": event.Type,
			"session_id": session.ID,
		})
		c.Status(http.StatusOK)
		return
	}

	packageID := parseUint(metadata["course_package_id"])
	if packageID == 0 {
		packageID = parseUint(metadata["package_id"])
	}
	userID := parseUint(metadata["user_id"])

	if packageID == 0 || userID == 0 {
		logger.Warn("Checkout webhook missing identifiers", map[string]interface{}{
			"request_id": requestID,
			"session_id": session.ID,
			"package_id": packageID,
			"user_id":    userID,
		})
		c.Status(http.StatusOK)
		return
	}

	req := models.GrantCoursePackageRequest{UserID: userID}
	if _, err := h.packageService.GrantToUser(packageID, req, 0); err != nil {
		logger.Error(err, "Failed to grant course access after checkout", map[string]interface{}{
			"request_id": requestID,
			"session_id": session.ID,
			"package_id": packageID,
			"user_id":    userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to grant course access"})
		return
	}

	logger.Info("Granted course access after Stripe checkout", map[string]interface{}{
		"request_id": requestID,
		"session_id": session.ID,
		"package_id": packageID,
		"user_id":    userID,
	})
	c.Status(http.StatusOK)
}

func parseUint(value string) uint {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	id, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil || id == 0 {
		return 0
	}
	return uint(id)
}
