package payments

import "context"

// Mode represents the type of checkout session that should be created.
type Mode string

const (
	// ModePayment processes a one-time payment for goods or services.
	ModePayment Mode = "payment"
)

// LineItem describes a purchasable item that should be included in a checkout session.
type LineItem struct {
	Name        string
	Description string
	AmountCents int64
	Quantity    int64
	Currency    string
}

// CheckoutParams encapsulates the parameters needed to create a checkout session.
type CheckoutParams struct {
	Mode          Mode
	SuccessURL    string
	CancelURL     string
	CustomerEmail string
	Metadata      map[string]string
	LineItems     []LineItem
}

// Session represents a checkout session created by a payment provider.
type Session struct {
	ID  string
	URL string
}

// SessionDetails represents the state of an existing checkout session retrieved from a payment provider.
type SessionDetails struct {
	ID            string
	Status        string
	PaymentStatus string
	Metadata      map[string]string
	CustomerEmail string
}

// Provider defines the behaviour required to create checkout sessions across payment vendors.
type Provider interface {
	CreateCheckoutSession(ctx context.Context, params CheckoutParams) (*Session, error)
	GetCheckoutSession(ctx context.Context, sessionID string) (*SessionDetails, error)
}
