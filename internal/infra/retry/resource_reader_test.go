package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

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

func (r *statusReader) Apply(context.Context, string) (domain.Resource, error) {
	r.applyCalls++
	return domain.Resource{}, r.applyErr
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
	wrapped, err := NewResourceReader(reader, &Policy{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Second}, clock, nil)
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
	reader := &statusReader{status: func(int) (domain.Resource, error) {
		return domain.Resource{}, app.Classify(app.ErrUnavailable, wantCause)
	}}
	wrapped, err := NewResourceReader(reader, &Policy{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Second}, &fakeClock{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = wrapped.Status(context.Background(), "example")
	if !errors.Is(err, app.ErrUnavailable) || !errors.Is(err, wantCause) || reader.statusCalls != 3 {
		t.Fatalf("Status() error = %v, attempts = %d, want unavailable cause after 3 attempts", err, reader.statusCalls)
	}
}

func TestResourceReaderDoesNotRetryApplyWithoutSafePolicy(t *testing.T) {
	wantErr := app.Classify(app.ErrUnavailable, errors.New("temporary outage"))
	reader := &statusReader{
		status:   func(int) (domain.Resource, error) { return domain.Resource{}, nil },
		applyErr: wantErr,
	}
	wrapped, err := NewResourceReader(reader, &Policy{MaxAttempts: 4, InitialDelay: time.Millisecond, MaxDelay: time.Second}, &fakeClock{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = wrapped.Apply(context.Background(), "example")
	if !errors.Is(err, wantErr) || reader.applyCalls != 1 {
		t.Fatalf("Apply() error = %v, attempts = %d, want one un-retried call", err, reader.applyCalls)
	}
}

func TestResourceReaderValidatesRetryPolicyAtConstruction(t *testing.T) {
	reader := &statusReader{status: func(int) (domain.Resource, error) { return domain.Resource{}, nil }}
	if _, err := NewResourceReader(reader, &Policy{MaxAttempts: 2}, &fakeClock{}, nil); err == nil {
		t.Fatal("NewResourceReader() error = nil, want invalid policy error")
	}
}
