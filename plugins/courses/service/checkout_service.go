package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/payments"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/logger"
)

var (
	// ErrCheckoutDisabled is returned when the checkout flow is not configured.
	ErrCheckoutDisabled = errors.New("course checkout is disabled")
	// ErrInvalidPackagePrice indicates that the course price cannot be processed by Stripe.
	ErrInvalidPackagePrice = errors.New("course package price must be greater than zero")
)

// CheckoutConfig defines configuration required to create checkout sessions.
type CheckoutConfig struct {
	SuccessURL string
	CancelURL  string
	Currency   string
}

// CheckoutSession wraps the information returned by the payment provider.
type CheckoutSession struct {
	ID  string
	URL string
}

// CheckoutService coordinates checkout session creation for course packages.
type CheckoutService struct {
	packageRepo repository.CoursePackageRepository
	provider    payments.Provider
	config      CheckoutConfig
}

// NewCheckoutService constructs a checkout service instance.
func NewCheckoutService(repo repository.CoursePackageRepository, provider payments.Provider, cfg CheckoutConfig) *CheckoutService {
	service := &CheckoutService{}
	service.SetConfig(cfg)
	service.SetDependencies(repo, provider)
	return service
}

// SetDependencies updates the repositories and payment provider used by the service.
func (s *CheckoutService) SetDependencies(repo repository.CoursePackageRepository, provider payments.Provider) {
	if s == nil {
		return
	}
	s.packageRepo = repo
	s.provider = provider
}

// SetConfig updates the checkout configuration used by the service.
func (s *CheckoutService) SetConfig(cfg CheckoutConfig) {
	if s == nil {
		return
	}
	s.config = normalizeCheckoutConfig(cfg)
}

// Enabled reports whether the checkout flow is ready for use.
func (s *CheckoutService) Enabled() bool {
	if s == nil {
		return false
	}
	cfg := s.config
	return s.packageRepo != nil && s.provider != nil && cfg.SuccessURL != "" && cfg.CancelURL != "" && cfg.Currency != ""
}

// Config returns a copy of the current checkout configuration.
func (s *CheckoutService) Config() CheckoutConfig {
	if s == nil {
		return CheckoutConfig{}
	}
	return s.config
}

// CreateCheckoutSession generates a checkout session for the requested course package.
func (s *CheckoutService) CreateCheckoutSession(ctx context.Context, req models.CourseCheckoutRequest) (*CheckoutSession, error) {
	if s == nil || !s.Enabled() {
		return nil, ErrCheckoutDisabled
	}

	logger.Info("Preparing checkout session", map[string]interface{}{
		"package_id": req.PackageID,
		"user_id":    req.UserID,
		"email":      strings.TrimSpace(req.CustomerEmail),
	})

	if req.PackageID == 0 {
		return nil, fmt.Errorf("course package id is required")
	}
	if req.UserID == 0 {
		return nil, fmt.Errorf("user id is required for checkout")
	}

	pkg, err := s.packageRepo.GetByID(req.PackageID)
	if err != nil {
		return nil, err
	}

	if pkg.PriceCents <= 0 {
		return nil, ErrInvalidPackagePrice
	}

	currency := s.config.Currency
	if currency == "" {
		return nil, ErrCheckoutDisabled
	}

	params := payments.CheckoutParams{
		Mode:       payments.ModePayment,
		SuccessURL: s.config.SuccessURL,
		CancelURL:  s.config.CancelURL,
		Metadata: map[string]string{
			"course_package_id":    strconv.FormatUint(uint64(pkg.ID), 10),
			"course_package_title": pkg.Title,
			"user_id":              strconv.FormatUint(uint64(req.UserID), 10),
		},
		LineItems: []payments.LineItem{
			{
				Name:        pkg.Title,
				Description: truncateDescription(pkg.Description),
				AmountCents: pkg.PriceCents,
				Quantity:    1,
				Currency:    currency,
			},
		},
	}

	if email := strings.TrimSpace(req.CustomerEmail); email != "" {
		params.CustomerEmail = email
	}

	if ctx == nil {
		ctx = context.Background()
	}

	session, err := s.provider.CreateCheckoutSession(ctx, params)
	if err != nil {
		logger.Error(err, "Failed to create checkout session with provider", map[string]interface{}{
			"package_id": req.PackageID,
			"user_id":    req.UserID,
		})
		return nil, err
	}

	logger.Info("Checkout session ready", map[string]interface{}{
		"package_id": req.PackageID,
		"user_id":    req.UserID,
		"session_id": session.ID,
	})

	return &CheckoutSession{ID: session.ID, URL: session.URL}, nil
}

func normalizeCheckoutConfig(cfg CheckoutConfig) CheckoutConfig {
	return CheckoutConfig{
		SuccessURL: strings.TrimSpace(cfg.SuccessURL),
		CancelURL:  strings.TrimSpace(cfg.CancelURL),
		Currency:   strings.ToLower(strings.TrimSpace(cfg.Currency)),
	}
}

func truncateDescription(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len([]rune(trimmed)) <= 500 {
		return trimmed
	}
	runes := []rune(trimmed)
	return string(runes[:500])
}

// Ensure the service satisfies gorm.ErrRecordNotFound propagation expectations.
var _ = gorm.ErrRecordNotFound
