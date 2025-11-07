package stripe

import "strings"

const (
	// SecretKeyPrefixStandard is the standard Stripe secret key prefix.
	SecretKeyPrefixStandard = "sk_"
	// SecretKeyPrefixRestricted is the Stripe restricted secret key prefix.
	SecretKeyPrefixRestricted = "rk_"
	// PublishableKeyPrefix is the Stripe publishable key prefix.
	PublishableKeyPrefix = "pk_"
	// WebhookSecretPrefix is the Stripe webhook signing secret prefix.
	WebhookSecretPrefix = "whsec_"
)

func hasAllowedPrefix(value string, prefixes ...string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}

	return false
}

// IsSecretKey reports whether the value looks like a Stripe secret key.
func IsSecretKey(value string) bool {
	return hasAllowedPrefix(value, SecretKeyPrefixStandard, SecretKeyPrefixRestricted)
}

// IsPublishableKey reports whether the value looks like a Stripe publishable key.
func IsPublishableKey(value string) bool {
	return hasAllowedPrefix(value, PublishableKeyPrefix)
}

// IsWebhookSecret reports whether the value looks like a Stripe webhook signing secret.
func IsWebhookSecret(value string) bool {
	return hasAllowedPrefix(value, WebhookSecretPrefix)
}
