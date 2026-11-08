package retry

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/domain"
	"github.com/turboBasic/template-go-cli-app/internal/logging"
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

type statusReader struct {
	statusCalls int
	status      func(int) (domain.Resource, error)
	applyCalls  int
	applyErr    error
}

func (r *statusReader) List(context.Context) ([]domain.Resource, error) { return nil, nil }

func (r *statusReader) Status(context.Context, string) (domain.Resource, error) {
	r.statusCalls++
	return r.status(r.statusCalls)
}

func (r *statusReader) Apply(context.Context, string, logging.Secret) (domain.Resource, error) {
	r.applyCalls++
	return domain.Resource{}, r.applyErr
}

func unavailable(cause error) *statusReader {
	return &statusReader{status: func(int) (domain.Resource, error) {
		return domain.Resource{}, app.Classify(app.ErrUnavailable, cause)
	}}
}

func TestResourceReaderRetriesStatusSuccessAfterTransientFailure(t *testing.T) {
	want := domain.Resource{Name: "example", Status: "ready"}
	reader := &statusReader{status: func(attempt int) (domain.Resource, error) {
		if attempt == 1 {
			return domain.Resource{}, app.Classify(app.ErrUnavailable, errors.New("temporary outage"))
		}
		return want, nil
	}}
	clock := &fakeClock{}
	wrapped, err := NewResourceReader(reader, Policy{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Second}, clock, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := wrapped.Status(context.Background(), want.Name)
	if err != nil || got != want || reader.statusCalls != 2 || len(clock.delays) != 1 {
		t.Fatalf("Status() = (%+v, %v), attempts = %d, waits = %v", got, err, reader.statusCalls, clock.delays)
	}
}

func TestResourceReaderStatusRetryExhaustion(t *testing.T) {
	wantCause := errors.New("temporary outage")
	reader := unavailable(wantCause)
	wrapped, err := NewResourceReader(reader, Policy{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Second}, &fakeClock{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = wrapped.Status(context.Background(), "example")
	if !errors.Is(err, app.ErrUnavailable) || !errors.Is(err, wantCause) || reader.statusCalls != 3 {
		t.Fatalf("Status() error = %v, attempts = %d, want unavailable cause after 3 attempts", err, reader.statusCalls)
	}
}

func TestResourceReaderStatusBackoffIsBoundedAndJittered(t *testing.T) {
	reader := unavailable(errors.New("temporary outage"))
	clock := &fakeClock{}
	random := &fakeRandom{values: []float64{0, 0.5, 1}}
	wrapped, err := NewResourceReader(reader, Policy{MaxAttempts: 4, InitialDelay: 100 * time.Millisecond, MaxDelay: 250 * time.Millisecond, Jitter: 0.5}, clock, random)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrapped.Status(context.Background(), "example"); !errors.Is(err, app.ErrUnavailable) || reader.statusCalls != 4 {
		t.Fatalf("Status() error = %v, attempts = %d, want unavailable after 4 attempts", err, reader.statusCalls)
	}
	if want := []time.Duration{50 * time.Millisecond, 200 * time.Millisecond, 250 * time.Millisecond}; !reflect.DeepEqual(clock.delays, want) {
		t.Fatalf("delays = %v, want %v", clock.delays, want)
	}
}

func TestResourceReaderStatusDoesNotRetryPermanentFailure(t *testing.T) {
	reader := &statusReader{status: func(int) (domain.Resource, error) { return domain.Resource{}, domain.ErrResourceNotFound }}
	wrapped, err := NewResourceReader(reader, Policy{MaxAttempts: 5, InitialDelay: time.Millisecond, MaxDelay: time.Second}, &fakeClock{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrapped.Status(context.Background(), "example"); !errors.Is(err, domain.ErrResourceNotFound) || reader.statusCalls != 1 {
		t.Fatalf("Status() error = %v, attempts = %d, want one un-retried call", err, reader.statusCalls)
	}
}

func TestResourceReaderStatusStopsPromptlyWhenCanceledDuringWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	clock := &fakeClock{sleep: func(ctx context.Context) error {
		cancel()
		return ctx.Err()
	}}
	reader := unavailable(errors.New("temporary outage"))
	wrapped, err := NewResourceReader(reader, Policy{MaxAttempts: 4, InitialDelay: time.Second, MaxDelay: time.Second}, clock, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = wrapped.Status(ctx, "example")
	if !errors.Is(err, context.Canceled) || !errors.Is(err, app.ErrUnavailable) || reader.statusCalls != 1 || len(clock.delays) != 1 {
		t.Fatalf("Status() error = %v, attempts = %d, waits = %v, want prompt cancellation keeping the cause", err, reader.statusCalls, clock.delays)
	}
}

func TestResourceReaderDoesNotRetryApply(t *testing.T) {
	wantErr := app.Classify(app.ErrUnavailable, errors.New("temporary outage"))
	reader := &statusReader{
		status:   func(int) (domain.Resource, error) { return domain.Resource{}, nil },
		applyErr: wantErr,
	}
	wrapped, err := NewResourceReader(reader, Policy{MaxAttempts: 4, InitialDelay: time.Millisecond, MaxDelay: time.Second}, &fakeClock{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = wrapped.Apply(context.Background(), "example", "")
	if !errors.Is(err, wantErr) || reader.applyCalls != 1 {
		t.Fatalf("Apply() error = %v, attempts = %d, want one un-retried call", err, reader.applyCalls)
	}
}
