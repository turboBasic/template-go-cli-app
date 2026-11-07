package output

import (
	"fmt"
	"io"
	"sync"

	"github.com/turboBasic/template-go-cli-app/internal/app"
)

// completionGate suppresses a completion event identical to the previous one.
type completionGate struct {
	lastEvent    app.ProgressEvent
	lastComplete bool
}

func isComplete(event app.ProgressEvent) bool {
	return event.Phase == "complete" || event.Total > 0 && event.Current >= event.Total
}

// skip reports whether event repeats the previous completion and records it otherwise.
func (g *completionGate) skip(event app.ProgressEvent) bool {
	complete := isComplete(event)
	if complete && g.lastComplete && event == g.lastEvent {
		return true
	}
	g.lastEvent = event
	g.lastComplete = complete
	return false
}

// PlainProgressReporter writes stable progress lines to its configured writer.
type PlainProgressReporter struct {
	mu     sync.Mutex
	writer io.Writer
	gate   completionGate
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

	if r.gate.skip(event) {
		return
	}
	_, _ = io.WriteString(r.writer, formatProgressEvent(event))
}

// StyledProgressReporter writes one styled line per progress event: a colored phase label with
// faint detail, and a check mark in place of the label on completion.
type StyledProgressReporter struct {
	mu     sync.Mutex
	writer io.Writer
	styles styles
	gate   completionGate
}

// NewStyledProgressReporter constructs a styled reporter for the given writer.
func NewStyledProgressReporter(writer io.Writer) *StyledProgressReporter {
	if writer == nil {
		writer = io.Discard
	}
	return &StyledProgressReporter{writer: writer, styles: newStyles()}
}

// Report writes one styled progress line, ignoring writer errors.
func (r *StyledProgressReporter) Report(event app.ProgressEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.gate.skip(event) {
		return
	}
	_, _ = io.WriteString(r.writer, r.styledLine(event))
}

// Close releases the reporter; it is safe to call more than once.
func (r *StyledProgressReporter) Close() {}

func (r *StyledProgressReporter) styledLine(event app.ProgressEvent) string {
	detail := event.Message + progressCounter(event)
	if isComplete(event) {
		return r.styles.ready.Render("✓") + " " + detail + "\n"
	}
	return r.styles.phase.Render(event.Phase) + " " + r.styles.detail.Render(detail) + "\n"
}

func progressCounter(event app.ProgressEvent) string {
	if event.Total > 0 {
		return fmt.Sprintf(" (%d/%d)", event.Current, event.Total)
	}
	return ""
}

// NoopProgressReporter discards progress events.
type NoopProgressReporter struct{}

// Report discards the progress event.
func (NoopProgressReporter) Report(app.ProgressEvent) {}

func formatProgressEvent(event app.ProgressEvent) string {
	return fmt.Sprintf("%s: %s", event.Phase, event.Message) + progressCounter(event) + "\n"
}
