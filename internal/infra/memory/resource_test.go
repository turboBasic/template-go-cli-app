package memory

import (
	"context"
	"reflect"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

func TestResourceReaderList(t *testing.T) {
	reader := ResourceReader{}
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

	if _, err := (ResourceReader{}).List(ctx); err == nil {
		t.Fatal("List() error = nil, want context cancellation")
	}
}
