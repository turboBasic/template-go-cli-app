package app

import (
	"context"
	"fmt"
	"time"
)

const (
	longRunningDuration = 30 * time.Second
	longRunningSteps    = 100
)

// WithLongRunningDuration replaces the length of the LongRunning demo; tests use it to avoid the wait.
func WithLongRunningDuration(duration time.Duration) ResourceOption {
	return func(s *ResourceService) { s.longRunning = duration }
}

// LongRunning is a demonstration of determinate progress: it counts longRunningSteps steps across the
// configured duration, reporting each one with Current and Total, then completion. A hundred steps of 1%
// keep the bar moving smoothly. A real application
// replaces it with its own counted work and deletes this demo.
func (s *ResourceService) LongRunning(ctx context.Context) error {
	s.logger.DebugContext(ctx, "running the long-running demo")
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("long-running demo: %w", classifyDeadline(err))
	}
	interval := s.longRunning / longRunningSteps
	for done := range longRunningSteps {
		s.reportProgress(ProgressEvent{Phase: "working", Current: done, Total: longRunningSteps, Message: "Working"})
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("long-running demo: %w", classifyDeadline(ctx.Err()))
		case <-timer.C:
		}
	}
	s.reportProgress(ProgressEvent{Phase: "complete", Current: longRunningSteps, Total: longRunningSteps, Message: "Long-running demo complete"})
	return nil
}
