package app

import (
	"context"
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

// Classify wraps err with a shared application error category. An error already in the
// category is returned unchanged, so classifying twice does not repeat the prefix.
func Classify(category, err error) error {
	if err == nil || errors.Is(err, category) {
		return err
	}
	return fmt.Errorf("%w: %w", category, err)
}

func classifyDeadline(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return Classify(ErrTimeout, err)
	}
	return err
}
