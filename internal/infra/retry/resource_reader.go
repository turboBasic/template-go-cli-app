package retry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/domain"
	"github.com/turboBasic/template-go-cli-app/internal/logging"
)

var _ app.ResourceReader = (*ResourceReader)(nil)

// Policy bounds attempts and exponential backoff. The configuration layer validates it.
type Policy struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Jitter       float64
}

// Clock waits for a delay or returns when the context is canceled.
type Clock interface {
	Sleep(context.Context, time.Duration) error
}

// RealClock waits using a cancelable timer.
type RealClock struct{}

// Sleep waits for delay unless ctx is canceled first.
func (RealClock) Sleep(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Random supplies a value in [0, 1) for jitter selection; *rand.Rand satisfies it.
type Random interface {
	Float64() float64
}

// ResourceReader retries only read-only status lookups; List and Apply are never retried.
type ResourceReader struct {
	reader   app.ResourceReader
	policy   Policy
	clock    Clock
	random   Random
	reporter app.ProgressReporter
}

// Option configures a ResourceReader.
type Option func(*ResourceReader)

// WithProgress reports each retry to reporter as an indeterminate "retrying" event.
func WithProgress(reporter app.ProgressReporter) Option {
	return func(r *ResourceReader) { r.reporter = reporter }
}

// NewResourceReader wraps a resource reader with status retries under policy.
func NewResourceReader(reader app.ResourceReader, policy Policy, clock Clock, random Random, options ...Option) (*ResourceReader, error) {
	if reader == nil {
		return nil, errors.New("resource reader is required")
	}
	if clock == nil || policy.Jitter > 0 && random == nil {
		return nil, errors.New("status retries require a clock and jitter requires a random source")
	}
	r := &ResourceReader{reader: reader, policy: policy, clock: clock, random: random}
	for _, option := range options {
		option(r)
	}
	return r, nil
}

// List delegates without retrying the operation.
func (r *ResourceReader) List(ctx context.Context) ([]domain.Resource, error) {
	return r.reader.List(ctx)
}

// Status retries only unavailable errors because this operation is read-only.
func (r *ResourceReader) Status(ctx context.Context, name string) (domain.Resource, error) {
	if r.policy.MaxAttempts <= 1 {
		return r.reader.Status(ctx, name)
	}
	var lastErr error
	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return domain.Resource{}, errors.Join(lastErr, err)
		}
		resource, err := r.reader.Status(ctx, name)
		if err == nil || !errors.Is(err, app.ErrUnavailable) {
			return resource, err
		}
		lastErr = err
		if attempt == r.policy.MaxAttempts {
			return domain.Resource{}, fmt.Errorf("retry exhausted after %d attempts: %w", attempt, lastErr)
		}
		delay := r.delay(attempt)
		if r.reporter != nil {
			r.reporter.Report(app.ProgressEvent{
				Phase:   "retrying",
				Message: fmt.Sprintf("Attempt %d of %d failed, retrying in %s", attempt, r.policy.MaxAttempts, delay),
			})
		}
		if err := r.clock.Sleep(ctx, delay); err != nil {
			return domain.Resource{}, errors.Join(lastErr, fmt.Errorf("wait before retry: %w", err))
		}
	}
}

// Apply delegates without retrying the mutating operation.
func (r *ResourceReader) Apply(ctx context.Context, name string, credential logging.Secret) (domain.Resource, error) {
	return r.reader.Apply(ctx, name, credential)
}

// delay is the exponential backoff after attempt, jittered and never above MaxDelay.
func (r *ResourceReader) delay(attempt int) time.Duration {
	delay := r.policy.InitialDelay
	for i := 1; i < attempt; i++ {
		if delay > r.policy.MaxDelay/2 {
			delay = r.policy.MaxDelay
			break
		}
		delay *= 2
	}
	if r.policy.Jitter > 0 {
		jittered := float64(delay) * (1 + r.policy.Jitter*(2*r.random.Float64()-1))
		delay = time.Duration(min(jittered, float64(r.policy.MaxDelay)))
	}
	return delay
}
