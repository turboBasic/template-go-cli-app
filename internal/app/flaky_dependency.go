package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

// FlakyFailures is how often the demo dependency fails before it succeeds: two failures, so a policy of
// three attempts succeeds and one of two is exhausted.
const FlakyFailures = 2

// StatusReader supplies the status of the dependency the FlakyDependency demo reads. It is only the read
// operation, so a decorator such as a retry policy can wrap it without carrying unrelated operations.
type StatusReader interface {
	Status(context.Context, string) (domain.Resource, error)
}

// WithFlakyDependency sets the dependency the FlakyDependency demo reads; the caller decides whether a
// retry policy wraps it.
func WithFlakyDependency(reader StatusReader) ResourceOption {
	return func(s *ResourceService) { s.flakyDependency = reader }
}

// FlakyDependency is a demonstration of retries: it reads the status of the dependency supplied by
// WithFlakyDependency, which in this template fails transiently FlakyFailures times per run and then
// succeeds. Without a retry policy around it the first failure is final. A real application replaces the
// simulated dependency with its own and deletes this demo.
func (s *ResourceService) FlakyDependency(ctx context.Context) (domain.Resource, error) {
	s.logger.DebugContext(ctx, "running the flaky-dependency demo")
	if err := ctx.Err(); err != nil {
		return domain.Resource{}, fmt.Errorf("flaky dependency: %w", classifyDeadline(err))
	}
	if s.flakyDependency == nil {
		return domain.Resource{}, fmt.Errorf("flaky dependency: %w", Classify(ErrValidation, errors.New("no dependency is configured")))
	}
	resource, err := s.flakyDependency.Status(ctx, "flaky-dependency")
	if err != nil {
		return domain.Resource{}, fmt.Errorf("flaky dependency: %w", classifyDeadline(err))
	}
	if err := resource.Validate(); err != nil {
		return domain.Resource{}, fmt.Errorf("flaky dependency: %w", Classify(ErrValidation, err))
	}
	// A retry leaves the spinner running; completion stops it before the caller prints the result.
	s.reportProgress(ProgressEvent{Phase: "complete", Message: "Dependency read"})
	return resource, nil
}
