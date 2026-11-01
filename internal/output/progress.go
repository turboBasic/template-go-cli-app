package output

import (
	"fmt"
	"io"
	"sync"

	"github.com/turboBasic/template-go-cli-app/internal/app"
)

// PlainProgressReporter writes stable progress lines to its configured writer.
type PlainProgressReporter struct {
	mu           sync.Mutex
	writer       io.Writer
	lastEvent    app.ProgressEvent
	lastComplete bool
}

// NewPlainProgressReporter constructs a plain reporter for the given writer.
func NewPlainProgressReporter(writer io.Writer) *PlainProgressReporter {
	if writer == nil {
		writer = io.Discard
	}
	return &PlainProgressReporter{writer: writer}
}

// Report writes one plain progress line, ignoring writer errors.
func (r *PlainProgressReporter) Report(event app.ProgressEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	complete := event.Phase == "complete" || event.Total > 0 && event.Current >= event.Total
	if complete && r.lastComplete && event == r.lastEvent {
		return
	}
	_, _ = io.WriteString(r.writer, formatProgressEvent(event))
	r.lastEvent = event
	r.lastComplete = complete
}

// NoopProgressReporter discards progress events.
type NoopProgressReporter struct{}

// Report discards the progress event.
func (NoopProgressReporter) Report(app.ProgressEvent) {}

func formatProgressEvent(event app.ProgressEvent) string {
	line := fmt.Sprintf("%s: %s", event.Phase, event.Message)
	if event.Total > 0 {
		line += fmt.Sprintf(" (%d/%d)", event.Current, event.Total)
	}
	return line + "\n"
}
