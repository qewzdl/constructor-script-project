package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// OptionalTime captures whether a timestamp field was provided in a request and its parsed value.
type OptionalTime struct {
	Set   bool       `json:"-"`
	Value *time.Time `json:"-"`
}

// UnmarshalJSON implements json.Unmarshaler, allowing the time field to accept RFC3339 strings or nulls.
func (ot *OptionalTime) UnmarshalJSON(data []byte) error {
	if ot == nil {
		return fmt.Errorf("optional time receiver is nil")
	}

	ot.Set = true

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || strings.EqualFold(trimmed, "null") {
		ot.Value = nil
		return nil
	}

	// Try to unmarshal as string first to allow flexible parsing.
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		parsed, err := parseTimeString(asString)
		if err != nil {
			return err
		}
		ot.Value = parsed
		return nil
	}

	var asTime time.Time
	if err := json.Unmarshal(data, &asTime); err != nil {
		return err
	}

	parsed := asTime.UTC()
	ot.Value = &parsed
	return nil
}

func parseTimeString(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, err
	}

	normalized := parsed.UTC()
	return &normalized, nil
}

// Or returns the stored value if explicitly set, otherwise it falls back to the provided default.
func (ot OptionalTime) Or(defaultValue *time.Time) *time.Time {
	if ot.Set {
		if ot.Value == nil {
			return nil
		}
		copy := ot.Value.UTC()
		return &copy
	}

	if defaultValue == nil {
		return nil
	}

	copy := defaultValue.UTC()
	return &copy
}

// Pointer returns the stored pointer regardless of whether the field was explicitly set.
func (ot OptionalTime) Pointer() *time.Time {
	if ot.Value == nil {
		return nil
	}
	copy := ot.Value.UTC()
	return &copy
}
