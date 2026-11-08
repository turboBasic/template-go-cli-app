package output

import (
	"fmt"
	"io"
	"sync"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
// mark in place of the label on completion. An event with a positive Total also shows a filled bar
// and percentage. Animated, it shows a spinner (or the bar, which glides to each new value) on the
// current phase from the first progress event until completion or Close; otherwise it writes one
// line per event.
type StyledProgressReporter struct {
	mu      sync.Mutex
	writer  io.Writer
	styles  styles
	bar     progress.Model
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
	return &StyledProgressReporter{writer: writer, styles: newStyles(), bar: newBar(), animate: animate}
}

// startProgram runs the spinner; a command that reports no progress never reaches it, so it never
// touches the terminal.
func (r *StyledProgressReporter) startProgram() {
	// The program owns neither input nor signals: main keeps SIGINT and the exit codes.
	r.program = tea.NewProgram(
		spinnerModel{spinner: spinner.New(spinner.WithStyle(r.styles.phase)), bar: r.bar, styles: r.styles},
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
		r.program.Send(labelFor(event))
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

type labelMsg struct {
	phase, detail  string
	current, total int
}

// labelFor converts an event into the spinner model's label, carrying the counts the bar needs.
func labelFor(event app.ProgressEvent) labelMsg {
	return labelMsg{phase: event.Phase, detail: event.Message + progressCounter(event), current: event.Current, total: event.Total}
}

type stopMsg struct{}

// spinnerModel draws the spinner line, or the bar while the current event has a total; it leaves
// nothing on screen once stopped.
type spinnerModel struct {
	spinner        spinner.Model
	bar            progress.Model
	styles         styles
	phase          string
	detail         string
	current, total int
	stopped        bool
}

func (m spinnerModel) Init() tea.Cmd { return m.spinner.Tick }

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case labelMsg:
		m.phase, m.detail, m.current, m.total = msg.phase, msg.detail, msg.current, msg.total
		if m.total > 0 {
			return m, m.bar.SetPercent(fraction(m.current, m.total))
		}
		return m, nil
	case progress.FrameMsg:
		var cmd tea.Cmd
		m.bar, cmd = m.bar.Update(msg)
		return m, cmd
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
	lead := m.spinner.View()
	if m.total > 0 {
		lead = m.bar.View()
	}
	return tea.NewView(lead + " " + m.styles.phase.Render(m.phase) + " " + m.styles.detail.Render(m.detail))
}

func (r *StyledProgressReporter) styledLine(event app.ProgressEvent) string {
	detail := event.Message + progressCounter(event)
	if isComplete(event) {
		return r.styles.ready.Render("✓") + " " + detail + "\n"
	}
	line := r.styles.phase.Render(event.Phase) + " " + r.styles.detail.Render(detail) + "\n"
	if event.Total > 0 {
		return r.bar.ViewAs(fraction(event.Current, event.Total)) + " " + line
	}
	return line
}

const (
	barCells     = 20
	percentCells = len(" 100%") // the model counts its percentage text in the width
)

// newBar returns a bar of barCells cells and a percentage, filled with the Bubbles default blend
// (purple to pink) so the colour shows how far the work has come. Full blocks keep one cell per
// step; the empty cells are a plain grey because the model colours them and cannot be Faint.
func newBar() progress.Model {
	bar := progress.New(progress.WithDefaultBlend(), progress.WithFillCharacters('█', '░'), progress.WithWidth(barCells+percentCells))
	bar.EmptyColor = lipgloss.Color("8")
	return bar
}

func fraction(current, total int) float64 { return float64(current) / float64(total) }

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
