package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestResourceServiceLongRunning(t *testing.T) {
	t.Run("counts every step and then completes", func(t *testing.T) {
		var events []ProgressEvent
		service := newTestResourceService(fakeResourceReader{}, progressRecorder{events: &events})
		WithLongRunningDuration(20 * time.Millisecond)(service)
		if err := service.LongRunning(context.Background()); err != nil {
			t.Fatalf("LongRunning() error = %v", err)
		}
		if len(events) != longRunningSteps+1 {
			t.Fatalf("events = %d, want %d", len(events), longRunningSteps+1)
		}
		for i, event := range events[:longRunningSteps] {
			if event.Phase != "working" || event.Current != i || event.Total != longRunningSteps {
				t.Fatalf("events[%d] = %#v, want working %d/%d", i, event, i, longRunningSteps)
			}
		}
		if last := events[longRunningSteps]; last.Phase != "complete" || last.Current != longRunningSteps || last.Total != longRunningSteps {
			t.Fatalf("last event = %#v, want complete %d/%[2]d", last, longRunningSteps)
		}
	})

	t.Run("stops between steps when canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var events []ProgressEvent
		service := newTestResourceService(fakeResourceReader{}, progressRecorder{events: &events, onReport: func(ProgressEvent) {
			if len(events) == 3 {
				cancel()
			}
		}})
		WithLongRunningDuration(20 * time.Millisecond)(service)
		if err := service.LongRunning(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("LongRunning() error = %v, want context cancellation", err)
		}
		if len(events) != 3 {
			t.Fatalf("events = %d, want 3 and no completion", len(events))
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
