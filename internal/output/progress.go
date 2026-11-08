package output

import (
	"fmt"
	"io"
	"sync"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/turboBasic/template-go-cli-app/internal/app"
)

func isComplete(event app.ProgressEvent) bool {
	return event.Phase == "complete" || event.Total > 0 && event.Current >= event.Total
}

// PlainProgressReporter writes plain, unanimated progress lines to its configured writer.
type PlainProgressReporter struct {
	mu     sync.Mutex
	writer io.Writer
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

	_, _ = io.WriteString(r.writer, formatProgressEvent(event))
}

// StyledProgressReporter shows styled progress: a colored phase label with faint detail and a check
// mark in place of the label on completion. Animated, it shows a spinner on the current phase from
// the first progress event until completion or Close; otherwise it writes one line per event.
type StyledProgressReporter struct {
	mu      sync.Mutex
	writer  io.Writer
	styles  styles
	animate bool         // false once stopped, so completion is never followed by a new spinner
	program *tea.Program // nil until the first progress event and after stopping
	done    chan struct{}
	closed  bool
}

// NewStyledProgressReporter constructs a styled reporter for the given writer. Animate shows a
// Bubble Tea spinner on the writer, which must therefore be a terminal; Close stops it.
func NewStyledProgressReporter(writer io.Writer, animate bool) *StyledProgressReporter {
	if writer == nil {
		writer = io.Discard
	}
	return &StyledProgressReporter{writer: writer, styles: newStyles(), animate: animate}
}

// startProgram runs the spinner; a command that reports no progress never reaches it, so it never
// touches the terminal.
func (r *StyledProgressReporter) startProgram() {
	// The program owns neither input nor signals: main keeps SIGINT and the exit codes.
	r.program = tea.NewProgram(
		spinnerModel{spinner: spinner.New(spinner.WithStyle(r.styles.phase)), styles: r.styles},
		tea.WithInput(nil), tea.WithoutSignalHandler(), tea.WithOutput(r.writer),
	)
	r.done = make(chan struct{})
	go func(program *tea.Program, done chan struct{}) {
		defer close(done)
		_, _ = program.Run()
	}(r.program, r.done)
}

// Report shows one progress event, ignoring writer errors. It does nothing after Close.
func (r *StyledProgressReporter) Report(event app.ProgressEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return
	}
	if r.animate && !isComplete(event) {
		if r.program == nil {
			r.startProgram()
		}
		r.program.Send(labelMsg{phase: event.Phase, detail: event.Message + progressCounter(event)})
		return
	}
	r.stopProgram()
	_, _ = io.WriteString(r.writer, r.styledLine(event))
}

// Close stops the spinner and restores the terminal; it is safe to call more than once.
func (r *StyledProgressReporter) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closed = true
	r.stopProgram()
}

func (r *StyledProgressReporter) stopProgram() {
	r.animate = false
	if r.program == nil {
		return
	}
	r.program.Send(stopMsg{})
	<-r.done
	r.program = nil
}

type labelMsg struct{ phase, detail string }

type stopMsg struct{}

// spinnerModel draws the spinner line; it leaves nothing on screen once stopped.
type spinnerModel struct {
	spinner spinner.Model
	styles  styles
	phase   string
	detail  string
	stopped bool
}

func (m spinnerModel) Init() tea.Cmd { return m.spinner.Tick }

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case labelMsg:
		m.phase, m.detail = msg.phase, msg.detail
		return m, nil
	case stopMsg:
		m.stopped = true
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() tea.View {
	if m.stopped {
		return tea.NewView("")
	}
	return tea.NewView(m.spinner.View() + " " + m.styles.phase.Render(m.phase) + " " + m.styles.detail.Render(m.detail))
}

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
