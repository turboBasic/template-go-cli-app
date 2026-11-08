package app

import (
	"context"
	"log/slog"
)

// Secret is a credential value that structured logging never prints.
type Secret string

// LogValue keeps the value out of structured logs whatever key it is logged under.
func (Secret) LogValue() slog.Value {
	return slog.StringValue("[REDACTED]")
}

// SecretResolver turns a provider reference into its secret value on demand.
// Implementers return errors carrying ErrValidation for a missing secret and
// ErrAuthentication for an authentication failure.
type SecretResolver interface {
	Resolve(ctx context.Context, reference string) (Secret, error)
}
