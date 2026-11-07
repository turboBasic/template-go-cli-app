package memory

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

func TestResourceReaderList(t *testing.T) {
	reader := NewResourceReader()
	want := []domain.Resource{
		{Name: "example-1", Status: "ready"},
		{Name: "example-2", Status: "pending"},
	}

	for range 2 {
		got, err := reader.List(context.Background())
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("List() = %#v, want %#v", got, want)
		}
		wantNames := []string{"example-1", "example-2"}
		if names := reader.Names(); !reflect.DeepEqual(names, wantNames) {
			t.Fatalf("Names() = %#v, want %#v", names, wantNames)
		}
	}
}

func TestResourceReaderListReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := NewResourceReader().List(ctx); err == nil {
		t.Fatal("List() error = nil, want context cancellation")
	}
}

func TestResourceReaderStatusAndApply(t *testing.T) {
	reader := NewResourceReader()
	resource, err := reader.Status(context.Background(), "example-2")
	if err != nil || resource.Status != "pending" {
		t.Fatalf("Status() = (%#v, %v), want pending resource", resource, err)
	}
	resource, err = reader.Apply(context.Background(), "example-2")
	if err != nil || resource.Status != "ready" {
		t.Fatalf("Apply() = (%#v, %v), want ready resource", resource, err)
	}
	resources, err := reader.List(context.Background())
	if err != nil || resources[1].Status != "ready" {
		t.Fatalf("List() after Apply() = (%#v, %v), want updated resource", resources, err)
	}
}

func TestResourceReaderRejectsUnknownResource(t *testing.T) {
	reader := NewResourceReader()
	if _, err := reader.Status(context.Background(), "missing"); err != domain.ErrResourceNotFound {
		t.Fatalf("Status() error = %v, want resource not found", err)
	}
	if _, err := reader.Apply(context.Background(), "missing"); err != domain.ErrResourceNotFound {
		t.Fatalf("Apply() error = %v, want resource not found", err)
	}
}

func TestResourceReaderApplyDelay(t *testing.T) {
	t.Run("waits for the delay", func(t *testing.T) {
		reader := NewResourceReader(WithApplyDelay(50 * time.Millisecond))
		start := time.Now()
		resource, err := reader.Apply(context.Background(), "example-2")
		if err != nil || resource.Status != "ready" {
			t.Fatalf("Apply() = %#v, %v; want ready", resource, err)
		}
		if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
			t.Fatalf("Apply() took %s, want at least 50ms", elapsed)
		}
	})

	t.Run("cancellation interrupts the wait and leaves the resource unchanged", func(t *testing.T) {
		reader := NewResourceReader(WithApplyDelay(time.Minute))
		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(10*time.Millisecond, cancel)
		start := time.Now()
		_, err := reader.Apply(ctx, "example-2")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Apply() error = %v, want context.Canceled", err)
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Fatalf("Apply() took %s after cancellation, want it to return promptly", elapsed)
		}
		if resource, err := reader.Status(context.Background(), "example-2"); err != nil || resource.Status != "pending" {
			t.Fatalf("Status() = %#v, %v; want pending", resource, err)
		}
	})
}
