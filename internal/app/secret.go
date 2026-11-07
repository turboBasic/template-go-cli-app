package app

import (
	"context"

	"github.com/turboBasic/template-go-cli-app/internal/logging"
)

// SecretResolver turns a provider reference into its secret value on demand.
// Implementers return errors carrying ErrValidation for a missing secret and
// ErrAuthentication for an authentication failure.
type SecretResolver interface {
	Resolve(ctx context.Context, reference string) (logging.Secret, error)
}
