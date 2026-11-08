package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestResourceServiceLongRunning(t *testing.T) {
	t.Run("reports counted progress and then completes", func(t *testing.T) {
		var events []ProgressEvent
		service := newTestResourceService(fakeResourceReader{}, progressRecorder{events: &events})
		WithLongRunningDuration(20 * time.Millisecond)(service)
		if err := service.LongRunning(context.Background()); err != nil {
			t.Fatalf("LongRunning() error = %v", err)
		}
		if len(events) < 2 {
			t.Fatalf("events = %d, want progress and completion", len(events))
		}
		previous := -1
		for i, event := range events[:len(events)-1] {
			if event.Phase != "working" || event.Current < previous || event.Total <= 0 || event.Current > event.Total {
				t.Fatalf("events[%d] = %#v, want monotonic counted progress", i, event)
			}
			previous = event.Current
		}
		if last := events[len(events)-1]; last.Phase != "complete" || last.Current != last.Total || last.Total <= 0 {
			t.Fatalf("last event = %#v, want complete counted progress", last)
		}
	})

	t.Run("stops between steps when canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var events []ProgressEvent
		service := newTestResourceService(fakeResourceReader{}, progressRecorder{events: &events, onReport: func(ProgressEvent) {
			if len(events) == 1 {
				cancel()
			}
		}})
		WithLongRunningDuration(20 * time.Millisecond)(service)
		if err := service.LongRunning(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("LongRunning() error = %v, want context cancellation", err)
		}
		if len(events) == 0 || events[len(events)-1].Phase == "complete" {
			t.Fatalf("events = %#v, want progress and no completion", events)
		}
	})

	t.Run("a canceled context reports nothing", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var events []ProgressEvent
		service := newTestResourceService(fakeResourceReader{}, progressRecorder{events: &events})
		if err := service.LongRunning(ctx); !errors.Is(err, context.Canceled) || len(events) != 0 {
			t.Fatalf("LongRunning() = %v with %d events, want cancellation and none", err, len(events))
		}
	})
}
