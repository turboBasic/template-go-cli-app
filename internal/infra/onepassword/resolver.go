// Package onepassword resolves op:// references through the 1Password CLI.
package onepassword

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/turboBasic/template-go-cli-app/internal/app"
)

const referencePrefix = "op://"

// Runner runs an external command and returns what it wrote to stdout and stderr.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)
}

// Resolver reads secrets with `op read`. Constructing it starts nothing: op is looked up only
// when a reference is resolved.
type Resolver struct {
	runner Runner
}

var _ app.SecretResolver = (*Resolver)(nil)

// NewResolver returns a Resolver that runs op through runner.
func NewResolver(runner Runner) *Resolver {
	return &Resolver{runner: runner}
}

// Resolve returns the secret behind an op:// reference. A missing vault, item or field is
// app.ErrValidation, an authentication failure is app.ErrAuthentication, cancellation is the
// context's error and anything else, a missing op binary included, is a plain error.
func (r *Resolver) Resolve(ctx context.Context, reference string) (app.Secret, error) {
	if !strings.HasPrefix(reference, referencePrefix) {
		return "", app.Classify(app.ErrValidation, fmt.Errorf("secret reference %q is not an %s reference", reference, referencePrefix))
	}
	// The reference is the only argument and is not a secret; the value comes back on stdout.
	stdout, stderr, err := r.runner.Run(ctx, "op", "read", "--no-newline", reference)
	if err == nil {
		return app.Secret(stdout), nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", ctxErr
	}
	if errors.Is(err, exec.ErrNotFound) {
		return "", fmt.Errorf("resolve %s: op is not installed: %w", reference, err)
	}
	// ponytail: op exits 1 for every failure, so classification matches its stderr text, which can
	// change between op versions; an unrecognised message stays a plain error. Upgrade path: a
	// contract test against a real op.
	missingMessages := []string{"isn't an item", "isn't a vault", "isn't a field", "isn't a section", "could not be found"}
	authenticationMessages := []string{"not currently signed in", "not signed in", "authorization prompt dismissed", "unauthorized", "authentication required"}
	message := strings.TrimSpace(string(stderr))
	switch {
	case matchesAny(message, missingMessages):
		return "", app.Classify(app.ErrValidation, fmt.Errorf("resolve %s: %s", reference, message))
	case matchesAny(message, authenticationMessages):
		return "", app.Classify(app.ErrAuthentication, fmt.Errorf("resolve %s: %s", reference, message))
	default:
		return "", fmt.Errorf("resolve %s with op: %w: %s", reference, err, message)
	}
}

func matchesAny(message string, needles []string) bool {
	message = strings.ToLower(message)
	for _, needle := range needles {
		if strings.Contains(message, needle) {
			return true
		}
	}
	return false
}
