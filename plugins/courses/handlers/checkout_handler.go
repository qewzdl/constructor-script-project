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

	baseFields := logContextFields(c)

	if h.packageService == nil {
		logger.Warn("Course checkout unavailable: package service missing", map[string]interface{}{
			"request_id": baseFields["request_id"],
			"path":       baseFields["path"],
		})
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "course package service unavailable"})
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		logger.Warn("Course checkout blocked: unauthenticated request", map[string]interface{}{
			"request_id": baseFields["request_id"],
			"path":       baseFields["path"],
			"method":     baseFields["method"],
			"client_ip":  baseFields["client_ip"],
		})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var req models.CourseCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Course checkout bad request", map[string]interface{}{
			"request_id": baseFields["request_id"],
			"path":       baseFields["path"],
			"method":     baseFields["method"],
			"client_ip":  baseFields["client_ip"],
			"error":      err.Error(),
		})
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
		logger.Info("Course checkout blocked: already owned", map[string]interface{}{
			"request_id": baseFields["request_id"],
			"user_id":    userID,
			"package_id": req.PackageID,
		})
		c.JSON(http.StatusConflict, gin.H{"error": "you already own this course"})
		return
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error(err, "Failed to check existing course access", map[string]interface{}{"package_id": req.PackageID, "user_id": userID})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to start checkout"})
		return
	}

	logger.Info("Starting course checkout session", map[string]interface{}{
		"request_id": baseFields["request_id"],
		"path":       baseFields["path"],
		"method":     baseFields["method"],
		"client_ip":  baseFields["client_ip"],
		"user_agent": baseFields["user_agent"],
		"user_id":    userID,
		"package_id": req.PackageID,
		"email":      strings.TrimSpace(req.CustomerEmail),
	})

	session, err := h.service.CreateCheckoutSession(c.Request.Context(), req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	logger.Info("Course checkout session created", map[string]interface{}{
		"request_id": baseFields["request_id"],
		"user_id":    userID,
		"package_id": req.PackageID,
		"session_id": session.ID,
	})

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
	baseFields := logContextFields(c)
	baseFields["webhook"] = "stripe_checkout"

	logger.Info("Received Stripe checkout webhook", map[string]interface{}{
		"request_id":  baseFields["request_id"],
		"content_len": c.Request.ContentLength,
		"user_agent":  c.Request.UserAgent(),
		"path":        baseFields["path"],
		"route":       baseFields["route"],
		"method":      baseFields["method"],
		"host":        baseFields["host"],
		"client_ip":   baseFields["client_ip"],
		"webhook":     baseFields["webhook"],
	})

	if h == nil || h.packageService == nil {
		logger.Warn("Course webhook unavailable: package service missing", map[string]interface{}{
			"request_id": baseFields["request_id"],
			"webhook":    baseFields["webhook"],
		})
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "course package service unavailable"})
		return
	}

	secret := strings.TrimSpace(h.webhookSecret)
	if secret == "" {
		logger.Warn("Course webhook secret not configured", map[string]interface{}{
			"request_id": baseFields["request_id"],
			"webhook":    baseFields["webhook"],
		})
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "stripe webhook not configured"})
		return
	}

	signature := strings.TrimSpace(c.GetHeader("Stripe-Signature"))
	if signature == "" {
		logger.Warn("Stripe webhook missing signature header", map[string]interface{}{
			"request_id": baseFields["request_id"],
			"webhook":    baseFields["webhook"],
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing stripe signature"})
		return
	}

	payload, err := c.GetRawData()
	if err != nil {
		logger.Error(err, "Failed to read Stripe webhook payload", map[string]interface{}{
			"request_id": baseFields["request_id"],
			"webhook":    baseFields["webhook"],
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read webhook payload"})
		return
	}

	if err := stripe.VerifyWebhookSignature(payload, signature, secret, 5*time.Minute); err != nil {
		logger.Warn("Invalid Stripe webhook signature", map[string]interface{}{
			"request_id": baseFields["request_id"],
			"error":      err.Error(),
			"webhook":    baseFields["webhook"],
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook signature"})
		return
	}

	var event stripeWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		logger.Warn("Invalid Stripe webhook payload", map[string]interface{}{
			"request_id": baseFields["request_id"],
			"error":      err.Error(),
			"webhook":    baseFields["webhook"],
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook payload"})
		return
	}

	if !strings.EqualFold(event.Type, "checkout.session.completed") {
		logger.Info("Ignored Stripe webhook event", map[string]interface{}{
			"request_id": baseFields["request_id"],
			"event_type": event.Type,
			"webhook":    baseFields["webhook"],
		})
		c.Status(http.StatusOK)
		return
	}

	session := event.Data.Object
	logger.Info("Stripe checkout session parsed", map[string]interface{}{
		"request_id":     baseFields["request_id"],
		"event_type":     event.Type,
		"session_id":     session.ID,
		"payment_status": session.PaymentStatus,
		"session_status": session.Status,
		"customer_email": session.CustomerEmail,
		"webhook":        baseFields["webhook"],
	})

	if strings.ToLower(strings.TrimSpace(session.PaymentStatus)) != "paid" && strings.ToLower(strings.TrimSpace(session.Status)) != "complete" {
		logger.Info("Checkout session not paid yet", map[string]interface{}{
			"request_id":     baseFields["request_id"],
			"event_type":     event.Type,
			"session_id":     session.ID,
			"payment_status": session.PaymentStatus,
			"session_status": session.Status,
			"customer_email": session.CustomerEmail,
			"webhook":        baseFields["webhook"],
		})
		c.Status(http.StatusOK)
		return
	}

	metadata := session.Metadata
	if len(metadata) == 0 {
		logger.Warn("Stripe checkout session missing metadata", map[string]interface{}{
			"request_id": baseFields["request_id"],
			"event_type": event.Type,
			"session_id": session.ID,
			"webhook":    baseFields["webhook"],
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
			"request_id": baseFields["request_id"],
			"session_id": session.ID,
			"package_id": packageID,
			"user_id":    userID,
			"metadata":   metadata,
			"webhook":    baseFields["webhook"],
		})
		c.Status(http.StatusOK)
		return
	}

	logger.Info("Attempting to grant course after checkout", map[string]interface{}{
		"request_id": baseFields["request_id"],
		"session_id": session.ID,
		"package_id": packageID,
		"user_id":    userID,
		"webhook":    baseFields["webhook"],
	})

	req := models.GrantCoursePackageRequest{UserID: userID}
	if _, err := h.packageService.GrantToUser(packageID, req, 0); err != nil {
		logger.Error(err, "Failed to grant course access after checkout", map[string]interface{}{
			"request_id": baseFields["request_id"],
			"session_id": session.ID,
			"package_id": packageID,
			"user_id":    userID,
			"webhook":    baseFields["webhook"],
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to grant course access"})
		return
	}

	logger.Info("Granted course access after Stripe checkout", map[string]interface{}{
		"request_id": baseFields["request_id"],
		"session_id": session.ID,
		"package_id": packageID,
		"user_id":    userID,
		"webhook":    baseFields["webhook"],
	})
	c.Status(http.StatusOK)
}

func logContextFields(c *gin.Context) map[string]interface{} {
	fields := map[string]interface{}{
		"path":       "",
		"route":      "",
		"method":     "",
		"client_ip":  "",
		"user_agent": "",
		"host":       "",
	}

	if c != nil && c.Request != nil {
		if c.Request.URL != nil {
			fields["path"] = c.Request.URL.Path
		}
		fields["method"] = c.Request.Method
		fields["user_agent"] = c.Request.UserAgent()
		fields["client_ip"] = c.ClientIP()
		fields["host"] = c.Request.Host
		fields["route"] = c.FullPath()
	}

	fields["request_id"] = strings.TrimSpace(c.GetString("request_id"))
	return fields
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
