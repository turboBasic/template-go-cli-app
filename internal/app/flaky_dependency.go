package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
	"github.com/turboBasic/template-go-cli-app/internal/logging"
)

// flakyFailures is how often the demo dependency fails before it succeeds: two failures, so a policy of
// three attempts succeeds and one of two is exhausted.
const flakyFailures = 2

// StatusRetries wraps a reader so its Status lookups are retried under the configured policy.
type StatusRetries func(ResourceReader) (ResourceReader, error)

// WithFlakyFailures replaces how often the FlakyDependency demo fails; tests use it to vary the count.
func WithFlakyFailures(failures int) ResourceOption {
	return func(s *ResourceService) { s.flakyFailures = failures }
}

// WithStatusRetries makes the FlakyDependency demo read through the wrapper retries returns.
func WithStatusRetries(retries StatusRetries) ResourceOption {
	return func(s *ResourceService) { s.statusRetries = retries }
}

// FlakyDependency is a demonstration of retries: it reads the status of a simulated dependency that fails
// transiently flakyFailures times and then succeeds. The dependency is built per call, so every run starts
// from its first failure. Without a retry policy the first failure is final. A real application replaces
// the simulated dependency with its own and deletes this demo.
func (s *ResourceService) FlakyDependency(ctx context.Context) (domain.Resource, error) {
	s.logger.DebugContext(ctx, "running the flaky-dependency demo")
	if err := ctx.Err(); err != nil {
		return domain.Resource{}, fmt.Errorf("flaky dependency: %w", classifyDeadline(err))
	}
	var reader ResourceReader = &flakyReader{failures: s.flakyFailures}
	if s.statusRetries != nil {
		var err error
		if reader, err = s.statusRetries(reader); err != nil {
			return domain.Resource{}, fmt.Errorf("flaky dependency: %w", err)
		}
	}
	resource, err := reader.Status(ctx, "flaky-dependency")
	if err != nil {
		return domain.Resource{}, fmt.Errorf("flaky dependency: %w", classifyDeadline(err))
	}
	if err := resource.Validate(); err != nil {
		return domain.Resource{}, fmt.Errorf("flaky dependency: %w", Classify(ErrValidation, err))
	}
	return resource, nil
}

// flakyReader fails its first failures Status calls with an unavailable error, then succeeds.
type flakyReader struct {
	failures int
	calls    int
}

func (r *flakyReader) Status(context.Context, string) (domain.Resource, error) {
	r.calls++
	if r.calls <= r.failures {
		return domain.Resource{}, Classify(ErrUnavailable, errors.New("simulated dependency is temporarily unavailable"))
	}
	return domain.Resource{Name: "flaky-dependency", Status: "ready"}, nil
}

func (*flakyReader) List(context.Context) ([]domain.Resource, error) {
	return nil, errors.New("the flaky-dependency demo does not support listing")
}

func (*flakyReader) Apply(context.Context, string, logging.Secret) (domain.Resource, error) {
	return domain.Resource{}, errors.New("the flaky-dependency demo does not support applying")
}
