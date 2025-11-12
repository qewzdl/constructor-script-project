package models

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// OptionalUint captures whether an unsigned integer field was provided in a request.
type OptionalUint struct {
	Set   bool
	Value *uint
}

// UnmarshalJSON implements json.Unmarshaler to detect if a field was supplied.
func (ou *OptionalUint) UnmarshalJSON(data []byte) error {
	if ou == nil {
		return fmt.Errorf("optional uint receiver is nil")
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return fmt.Errorf("optional uint cannot parse empty input")
	}
	ou.Set = true
	if bytes.EqualFold(trimmed, []byte("null")) {
		ou.Value = nil
		return nil
	}
	var value uint64
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return err
	}
	converted := uint(value)
	ou.Value = &converted
	return nil
}

// Pointer returns the parsed pointer if the field was set, otherwise nil.
func (ou OptionalUint) Pointer() *uint {
	if !ou.Set {
		return nil
	}
	return ou.Value
}

// Or returns the parsed pointer or a default value if unset or null.
func (ou OptionalUint) Or(defaultValue *uint) *uint {
	if ou.Value != nil {
		return ou.Value
	}
	return defaultValue
}
