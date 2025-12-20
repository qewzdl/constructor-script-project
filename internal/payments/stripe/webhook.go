package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// VerifyWebhookSignature validates a Stripe webhook signature header against the payload.
// It follows Stripe's recommendation: https://stripe.com/docs/webhooks/signatures
func VerifyWebhookSignature(payload []byte, header, secret string, tolerance time.Duration) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return errors.New("stripe webhook secret is required")
	}

	timestamp, signatures := parseSignatureHeader(header)
	if timestamp == "" || len(signatures) == 0 {
		return errors.New("stripe signature header is missing required fields")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid stripe signature timestamp: %w", err)
	}

	if tolerance > 0 {
		now := time.Now().Unix()
		diff := now - ts
		if diff < 0 {
			diff = -diff
		}
		if diff > int64(tolerance.Seconds()) {
			return errors.New("stripe signature timestamp outside tolerance")
		}
	}

	signedPayload := timestamp + "." + string(payload)
	expectedMAC := computeHMACSHA256([]byte(signedPayload), []byte(secret))

	for _, sig := range signatures {
		decoded, err := hex.DecodeString(sig)
		if err != nil {
			continue
		}
		if hmac.Equal(decoded, expectedMAC) {
			return nil
		}
	}

	return errors.New("no matching stripe signature found")
}

func parseSignatureHeader(header string) (string, []string) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", nil
	}

	var (
		timestamp  string
		signatures []string
	)

	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, "t="):
			timestamp = strings.TrimPrefix(part, "t=")
		case strings.HasPrefix(part, "v1="):
			if sig := strings.TrimPrefix(part, "v1="); sig != "" {
				signatures = append(signatures, sig)
			}
		}
	}

	return timestamp, signatures
}

func computeHMACSHA256(message, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}
