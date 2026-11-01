package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Policy bounds attempts and exponential backoff for one operation.
type Policy struct {
	MaxAttempts  int           `koanf:"max_attempts"`
	InitialDelay time.Duration `koanf:"initial_delay"`
	MaxDelay     time.Duration `koanf:"max_delay"`
	Jitter       float64       `koanf:"jitter"`
}

// Validate checks that a retry policy is bounded and usable.
func (p Policy) Validate() error {
	if p.MaxAttempts < 1 {
		return fmt.Errorf("max attempts must be positive")
	}
	if p.InitialDelay < 0 || p.MaxDelay < 0 {
		return fmt.Errorf("retry delays must be non-negative")
	}
	if p.MaxAttempts > 1 && (p.InitialDelay == 0 || p.MaxDelay < p.InitialDelay) {
		return fmt.Errorf("retry delays must satisfy 0 < initial delay <= max delay")
	}
	if math.IsNaN(p.Jitter) || p.Jitter < 0 || p.Jitter > 1 {
		return fmt.Errorf("jitter must be between 0 and 1")
	}
	return nil
}

// Safety records the operation's retry guarantee. The zero value is unsafe.
type Safety uint8

const (
	// Unspecified is the safe default when no retry guarantee was provided.
	Unspecified Safety = iota
	// Idempotent marks an operation that can be repeated without changing its effect.
	Idempotent
)

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

// Random supplies a value in the inclusive range [0, 1] for jitter selection.
type Random interface {
	Float64() float64
}

// RandomSource is a per-operation pseudo-random source for retry jitter.
type RandomSource struct {
	random *rand.Rand
}

// NewRandomSource constructs an independent random source with seed.
func NewRandomSource(seed int64) *RandomSource {
	return &RandomSource{random: rand.New(rand.NewSource(seed))}
}

// Float64 returns the next pseudo-random value in [0, 1).
func (r *RandomSource) Float64() float64 { return r.random.Float64() }

// Retryable reports whether an operation error may be retried.
type Retryable func(error) bool

// Operation is a context-aware infrastructure operation.
type Operation func(context.Context) error

// Do runs operation with bounded retries only when policy, safety and error classification opt in.
func Do(ctx context.Context, operation Operation, policy *Policy, safety Safety, retryable Retryable, clock Clock, random Random) error {
	if operation == nil {
		return errors.New("retry operation is required")
	}
	if policy == nil {
		return operation(ctx)
	}
	if err := policy.Validate(); err != nil {
		return fmt.Errorf("validate retry policy: %w", err)
	}
	if safety != Idempotent || policy.MaxAttempts == 1 || retryable == nil {
		return operation(ctx)
	}
	if clock == nil {
		return errors.New("retry clock is required")
	}
	if policy.Jitter > 0 && random == nil {
		return errors.New("retry random source is required when jitter is enabled")
	}

	var lastErr error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return errors.Join(lastErr, err)
		}
		lastErr = operation(ctx)
		if lastErr == nil || !retryable(lastErr) {
			return lastErr
		}
		if attempt == policy.MaxAttempts {
			return fmt.Errorf("retry exhausted after %d attempts: %w", attempt, lastErr)
		}
		delay := backoff(*policy, attempt)
		if policy.Jitter > 0 {
			sample := random.Float64()
			if math.IsNaN(sample) || sample < 0 || sample > 1 {
				return errors.Join(lastErr, fmt.Errorf("retry random value %v is outside [0, 1]", sample))
			}
			jittered := float64(delay) * (1 + policy.Jitter*(2*sample-1))
			if jittered > float64(policy.MaxDelay) {
				delay = policy.MaxDelay
			} else {
				delay = time.Duration(jittered)
			}
		}
		if err := clock.Sleep(ctx, delay); err != nil {
			return errors.Join(lastErr, fmt.Errorf("wait before retry: %w", err))
		}
	}
	return lastErr
}

func backoff(policy Policy, attempt int) time.Duration {
	delay := policy.InitialDelay
	for i := 1; i < attempt && delay < policy.MaxDelay; i++ {
		if delay > policy.MaxDelay/2 {
			return policy.MaxDelay
		}
		delay *= 2
	}
	if delay > policy.MaxDelay {
		return policy.MaxDelay
	}
	return delay
}
