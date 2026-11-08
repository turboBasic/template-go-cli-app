package app

import (
	"context"
	"errors"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

func TestFlakyDependency(t *testing.T) {
	want := domain.Resource{Name: "flaky-dependency", Status: "ready"}

	t.Run("no failures succeeds without a wrapper", func(t *testing.T) {
		service := newTestResourceService(fakeResourceReader{})
		WithFlakyFailures(0)(service)
		if got, err := service.FlakyDependency(context.Background()); err != nil || got != want {
			t.Fatalf("FlakyDependency() = (%+v, %v), want %+v", got, err, want)
		}
	})

	t.Run("a failure without a wrapper is final", func(t *testing.T) {
		service := newTestResourceService(fakeResourceReader{})
		WithFlakyFailures(1)(service)
		if _, err := service.FlakyDependency(context.Background()); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("FlakyDependency() error = %v, want ErrUnavailable", err)
		}
	})

	t.Run("a wrapper sees the default failures and then the resource", func(t *testing.T) {
		var errs []error
		service := newTestResourceService(fakeResourceReader{})
		WithStatusRetries(func(reader ResourceReader) (ResourceReader, error) {
			var got domain.Resource
			for range flakyFailures + 1 {
				var err error
				got, err = reader.Status(context.Background(), "ignored")
				errs = append(errs, err)
			}
			return fakeResourceReader{resource: got}, nil
		})(service)
		if got, err := service.FlakyDependency(context.Background()); err != nil || got != want {
			t.Fatalf("FlakyDependency() = (%+v, %v), want %+v", got, err, want)
		}
		if len(errs) != 3 || !errors.Is(errs[0], ErrUnavailable) || !errors.Is(errs[1], ErrUnavailable) || errs[2] != nil {
			t.Fatalf("Status() errors = %v, want two ErrUnavailable then nil", errs)
		}
	})

	t.Run("a wrapper that cannot be built fails the demo", func(t *testing.T) {
		wantErr := errors.New("bad policy")
		service := newTestResourceService(fakeResourceReader{})
		WithStatusRetries(func(ResourceReader) (ResourceReader, error) { return nil, wantErr })(service)
		if _, err := service.FlakyDependency(context.Background()); !errors.Is(err, wantErr) {
			t.Fatalf("FlakyDependency() error = %v, want %v", err, wantErr)
		}
	})

	t.Run("a canceled context never reaches the dependency", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		called := false
		service := newTestResourceService(fakeResourceReader{})
		WithStatusRetries(func(reader ResourceReader) (ResourceReader, error) {
			called = true
			return reader, nil
		})(service)
		if _, err := service.FlakyDependency(ctx); !errors.Is(err, context.Canceled) || called {
			t.Fatalf("FlakyDependency() error = %v, wrapper called = %t, want cancellation and no call", err, called)
		}
	})
}
