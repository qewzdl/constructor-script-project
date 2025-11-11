package service

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

var errValidation = errors.New("courseservice: validation error")

type validationError struct {
	message string
}

func (e *validationError) Error() string {
	return e.message
}

func (e *validationError) Unwrap() error {
	return errValidation
}

func newValidationError(format string, args ...interface{}) error {
	message := strings.TrimSpace(fmt.Sprintf(format, args...))
	if message == "" {
		message = "invalid input"
	}
	return &validationError{message: message}
}

// IsValidationError reports whether the provided error indicates invalid user input.
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, errValidation)
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	var sqlState interface{ SQLState() string }
	if errors.As(err, &sqlState) {
		return sqlState.SQLState() == "23505"
	}

	return false
}
