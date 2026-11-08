package memory

import (
	"context"
	"errors"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

// FlakyReader simulates a dependency that fails its first failures Status calls with an unavailable error
// and then succeeds. Construct one per run so every run starts from its first failure.
type FlakyReader struct {
	failures int
	calls    int
}

var _ app.StatusReader = (*FlakyReader)(nil)

// NewFlakyReader constructs a reader that fails failures times before it succeeds.
func NewFlakyReader(failures int) *FlakyReader {
	return &FlakyReader{failures: failures}
}

// Status fails with an unavailable error until the configured failures are used up.
func (r *FlakyReader) Status(context.Context, string) (domain.Resource, error) {
	r.calls++
	if r.calls <= r.failures {
		return domain.Resource{}, app.Classify(app.ErrUnavailable, errors.New("simulated dependency is temporarily unavailable"))
	}
	return domain.Resource{Name: "flaky-dependency", Status: "ready"}, nil
}
