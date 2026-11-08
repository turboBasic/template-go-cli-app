package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/app"
)

func TestFlakyReaderFailsThenSucceeds(t *testing.T) {
	reader := NewFlakyReader(2)
	for call := 1; call <= 2; call++ {
		if _, err := reader.Status(context.Background(), "x"); !errors.Is(err, app.ErrUnavailable) {
			t.Fatalf("call %d error = %v, want ErrUnavailable", call, err)
		}
	}
	if got, err := reader.Status(context.Background(), "x"); err != nil || got.Status != "ready" {
		t.Fatalf("call 3 = (%+v, %v), want a ready resource", got, err)
	}
}
