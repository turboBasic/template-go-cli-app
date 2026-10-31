package memory

import (
	"context"
	"reflect"
	"testing"

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
