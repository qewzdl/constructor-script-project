package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"constructor-script-backend/internal/payments"
)

const defaultAPIBase = "https://api.stripe.com"

// Provider implements the payments.Provider interface for Stripe Checkout using direct HTTP calls.
type Provider struct {
	secretKey  string
	httpClient *http.Client
	apiBaseURL string
	userAgent  string
}

// NewProvider constructs a Stripe provider using the supplied secret API key.
func NewProvider(secretKey string) (*Provider, error) {
	key := strings.TrimSpace(secretKey)
	if key == "" {
		return nil, errors.New("stripe secret key is required")
	}

	return &Provider{
		secretKey:  key,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		apiBaseURL: defaultAPIBase,
		userAgent:  "constructor-script-backend/stripe-checkout",
	}, nil
}

func (p *Provider) createRequest(ctx context.Context, params payments.CheckoutParams) (*http.Request, error) {
	form := url.Values{}
	mode := params.Mode
	if mode == "" {
		mode = payments.ModePayment
	}
	form.Set("mode", string(mode))
	form.Set("success_url", params.SuccessURL)
	form.Set("cancel_url", params.CancelURL)

	if email := strings.TrimSpace(params.CustomerEmail); email != "" {
		form.Set("customer_email", email)
	}

	for key, value := range params.Metadata {
		if key == "" || value == "" {
			continue
		}
		form.Set("metadata["+key+"]", value)
	}

	if len(params.LineItems) == 0 {
		return nil, errors.New("at least one line item is required")
	}

	for index, item := range params.LineItems {
		if item.AmountCents <= 0 {
			return nil, fmt.Errorf("line item %q has invalid amount", item.Name)
		}
		currency := strings.ToLower(strings.TrimSpace(item.Currency))
		if currency == "" {
			return nil, fmt.Errorf("line item %q currency is required", item.Name)
		}

		quantity := item.Quantity
		if quantity <= 0 {
			quantity = 1
		}

		prefix := fmt.Sprintf("line_items[%d]", index)
		form.Set(prefix+"[quantity]", strconv.FormatInt(quantity, 10))
		form.Set(prefix+"[price_data][currency]", currency)
		form.Set(prefix+"[price_data][unit_amount]", strconv.FormatInt(item.AmountCents, 10))
		form.Set(prefix+"[price_data][product_data][name]", item.Name)
		if desc := strings.TrimSpace(item.Description); desc != "" {
			form.Set(prefix+"[price_data][product_data][description]", desc)
		}
	}

	endpoint := fmt.Sprintf("%s/v1/checkout/sessions", strings.TrimRight(p.apiBaseURL, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+p.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", p.userAgent)

	return req, nil
}

// CreateCheckoutSession creates a Stripe Checkout session for the provided purchase parameters.
func (p *Provider) CreateCheckoutSession(ctx context.Context, params payments.CheckoutParams) (*payments.Session, error) {
	if p == nil {
		return nil, errors.New("stripe provider is not configured")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	req, err := p.createRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload struct {
		ID    string `json:"id"`
		URL   string `json:"url"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("stripe response decode failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		message := strings.TrimSpace(payload.Error.Message)
		if message == "" {
			message = fmt.Sprintf("stripe returned status %d", resp.StatusCode)
		}
		return nil, errors.New(message)
	}

	if payload.ID == "" || payload.URL == "" {
		return nil, errors.New("stripe response missing session details")
	}

	return &payments.Session{ID: payload.ID, URL: payload.URL}, nil
}
