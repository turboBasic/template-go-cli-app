package app

import (
	"context"
	"errors"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

type statusFunc func(context.Context, string) (domain.Resource, error)

func (f statusFunc) Status(ctx context.Context, name string) (domain.Resource, error) {
	return f(ctx, name)
}

func TestFlakyDependency(t *testing.T) {
	want := domain.Resource{Name: "flaky-dependency", Status: "ready"}
	ready := statusFunc(func(context.Context, string) (domain.Resource, error) { return want, nil })

	t.Run("a ready dependency succeeds", func(t *testing.T) {
		service := newTestResourceService(fakeResourceReader{})
		WithFlakyDependency(ready)(service)
		if got, err := service.FlakyDependency(context.Background()); err != nil || got != want {
			t.Fatalf("FlakyDependency() = (%+v, %v), want %+v", got, err, want)
		}
	})

	t.Run("success ends progress so a spinner left by a retry stops", func(t *testing.T) {
		var events []ProgressEvent
		service := newTestResourceService(fakeResourceReader{}, progressRecorder{events: &events})
		WithFlakyDependency(ready)(service)
		if _, err := service.FlakyDependency(context.Background()); err != nil {
			t.Fatalf("FlakyDependency() error = %v", err)
		}
		if len(events) != 1 || events[0].Phase != "complete" {
			t.Fatalf("events = %#v, want one complete event", events)
		}
	})

	t.Run("a dependency failure is returned", func(t *testing.T) {
		service := newTestResourceService(fakeResourceReader{})
		WithFlakyDependency(statusFunc(func(context.Context, string) (domain.Resource, error) {
			return domain.Resource{}, Classify(ErrUnavailable, errors.New("down"))
		}))(service)
		if _, err := service.FlakyDependency(context.Background()); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("FlakyDependency() error = %v, want ErrUnavailable", err)
		}
	})

	t.Run("a missing dependency is a validation error", func(t *testing.T) {
		service := newTestResourceService(fakeResourceReader{})
		if _, err := service.FlakyDependency(context.Background()); !errors.Is(err, ErrValidation) {
			t.Fatalf("FlakyDependency() error = %v, want ErrValidation", err)
		}
	})

	t.Run("a canceled context never reaches the dependency", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		called := false
		service := newTestResourceService(fakeResourceReader{})
		WithFlakyDependency(statusFunc(func(context.Context, string) (domain.Resource, error) {
			called = true
			return want, nil
		}))(service)
		if _, err := service.FlakyDependency(ctx); !errors.Is(err, context.Canceled) || called {
			t.Fatalf("FlakyDependency() error = %v, dependency called = %t, want cancellation and no call", err, called)
		}
	})
}
