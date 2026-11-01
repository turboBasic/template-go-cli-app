package app

import (
	"errors"
	"fmt"
)

// Error categories describe application failures without hiding their causes.
var (
	ErrValidation     = errors.New("validation error")
	ErrAuthentication = errors.New("authentication error")
	ErrAuthorization  = errors.New("authorization error")
	ErrConflict       = errors.New("conflict error")
	ErrUnavailable    = errors.New("unavailable error")
	ErrTimeout        = errors.New("timeout error")
	ErrCancellation   = errors.New("cancellation error")
)

// Classify wraps err with a shared application error category.
func Classify(category, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", category, err)
}
