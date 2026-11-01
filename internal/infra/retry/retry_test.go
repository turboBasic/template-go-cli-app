package retry

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeClock struct {
	delays []time.Duration
	sleep  func(context.Context) error
}

func (c *fakeClock) Sleep(ctx context.Context, delay time.Duration) error {
	c.delays = append(c.delays, delay)
	if c.sleep != nil {
		return c.sleep(ctx)
	}
	return nil
}

type fakeRandom struct{ values []float64 }

func (r *fakeRandom) Float64() float64 {
	value := r.values[0]
	r.values = r.values[1:]
	return value
}

func TestDoRetriesWithBoundedBackoffAndJitter(t *testing.T) {
	wantErr := errors.New("temporarily unavailable")
	clock := &fakeClock{}
	random := &fakeRandom{values: []float64{0, 0.5, 1}}
	attempts := 0
	err := Do(context.Background(), func(context.Context) error {
		attempts++
		return wantErr
	}, &Policy{MaxAttempts: 4, InitialDelay: 100 * time.Millisecond, MaxDelay: 250 * time.Millisecond, Jitter: 0.5}, Idempotent, func(error) bool { return true }, clock, random)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Do() error = %v, want wrapped operation error", err)
	}
	if attempts != 4 {
		t.Fatalf("attempts = %d, want 4", attempts)
	}
	if want := []time.Duration{50 * time.Millisecond, 200 * time.Millisecond, 250 * time.Millisecond}; !reflect.DeepEqual(clock.delays, want) {
		t.Fatalf("delays = %v, want %v", clock.delays, want)
	}
}

func TestDoStopsAfterTransientThenSuccess(t *testing.T) {
	wantErr := errors.New("temporarily unavailable")
	clock := &fakeClock{}
	attempts := 0
	err := Do(context.Background(), func(context.Context) error {
		attempts++
		if attempts == 1 {
			return wantErr
		}
		return nil
	}, &Policy{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Second}, Idempotent, func(err error) bool { return errors.Is(err, wantErr) }, clock, nil)
	if err != nil || attempts != 2 || len(clock.delays) != 1 {
		t.Fatalf("Do() = (%v, %d attempts, %v delays), want success after one retry", err, attempts, clock.delays)
	}
}

func TestDoDoesNotRetryUnapprovedOrPermanentFailures(t *testing.T) {
	for _, test := range []struct {
		name      string
		safety    Safety
		retryable Retryable
	}{
		{name: "operation not approved", safety: Unspecified, retryable: func(error) bool { return true }},
		{name: "permanent error", safety: Idempotent, retryable: func(error) bool { return false }},
	} {
		t.Run(test.name, func(t *testing.T) {
			attempts := 0
			err := Do(context.Background(), func(context.Context) error {
				attempts++
				return errors.New("failed")
			}, &Policy{MaxAttempts: 5, InitialDelay: time.Millisecond, MaxDelay: time.Second}, test.safety, test.retryable, &fakeClock{}, &fakeRandom{})
			if err == nil || attempts != 1 {
				t.Fatalf("Do() = (%v, %d attempts), want one failed attempt", err, attempts)
			}
		})
	}
}

func TestDoStopsPromptlyWhenCanceledDuringWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	clock := &fakeClock{sleep: func(ctx context.Context) error {
		cancel()
		return ctx.Err()
	}}
	err := Do(ctx, func(context.Context) error {
		attempts++
		return errors.New("temporarily unavailable")
	}, &Policy{MaxAttempts: 4, InitialDelay: time.Second, MaxDelay: time.Second}, Idempotent, func(error) bool { return true }, clock, &fakeRandom{})
	if !errors.Is(err, context.Canceled) || attempts != 1 || len(clock.delays) != 1 {
		t.Fatalf("Do() = (%v, %d attempts, %v delays), want prompt cancellation during first wait", err, attempts, clock.delays)
	}
}

func TestPolicyValidate(t *testing.T) {
	for _, policy := range []Policy{
		{MaxAttempts: 0},
		{MaxAttempts: 2, InitialDelay: time.Second, MaxDelay: time.Millisecond},
		{MaxAttempts: 2, InitialDelay: time.Second, MaxDelay: time.Second, Jitter: 1.1},
	} {
		if err := policy.Validate(); err == nil {
			t.Errorf("Validate(%+v) = nil, want invalid-policy error", policy)
		}
	}
}
