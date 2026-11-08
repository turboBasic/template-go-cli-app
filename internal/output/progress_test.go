package output

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	"github.com/charmbracelet/harmonica"
	"github.com/turboBasic/template-go-cli-app/internal/app"
)

func TestPlainProgressReporter(t *testing.T) {
	tests := []struct {
		name   string
		events []app.ProgressEvent
		want   string
	}{
		{
			name: "phase changes and redirected writer",
			events: []app.ProgressEvent{
				{Phase: "validating", Message: "Checking resource"},
				{Phase: "applying", Message: "Applying resource"},
			},
			want: "validating: Checking resource\napplying: Applying resource\n",
		},
		{
			name: "measurable and indeterminate progress",
			events: []app.ProgressEvent{
				{Phase: "applying", Current: 1, Total: 2, Message: "Applying"},
				{Phase: "waiting", Current: 4, Message: "Waiting"},
			},
			want: "applying: Applying (1/2)\nwaiting: Waiting\n",
		},
		{
			name: "identical completions from separate operations are both reported",
			events: []app.ProgressEvent{
				{Phase: "applying", Current: 2, Total: 2, Message: "Applied"},
				{Phase: "applying", Current: 2, Total: 2, Message: "Applied"},
			},
			want: "applying: Applied (2/2)\napplying: Applied (2/2)\n",
		},
		{
			name: "completion phase is retained after other progress",
			events: []app.ProgressEvent{
				{Phase: "applying", Message: "Applying"},
				{Phase: "complete", Message: "Done"},
			},
			want: "applying: Applying\ncomplete: Done\n",
		},
		{
			name:   "writer errors are ignored",
			events: []app.ProgressEvent{{Phase: "applying", Message: "Applying"}},
			want:   "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var writer bytes.Buffer
			var destination interface{ Write([]byte) (int, error) } = &writer
			if test.name == "writer errors are ignored" {
				destination = failingWriter{}
			}
			reporter := NewPlainProgressReporter(destination)
			for _, event := range test.events {
				reporter.Report(event)
			}
			if got := writer.String(); got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
			if strings.Contains(writer.String(), "\x1b[") {
				t.Fatalf("output contains ANSI escape: %q", writer.String())
			}
		})
	}
}

func TestPlainProgressReporterSerializesConcurrentCalls(t *testing.T) {
	var output bytes.Buffer
	reporter := NewPlainProgressReporter(&output)
	const calls = 100
	var workers sync.WaitGroup
	workers.Add(calls)
	for range calls {
		go func() {
			defer workers.Done()
			reporter.Report(app.ProgressEvent{Phase: "working", Message: "Update"})
		}()
	}
	workers.Wait()
	if got := strings.Count(output.String(), "working: Update\n"); got != calls {
		t.Fatalf("reported lines = %d, want %d", got, calls)
	}
}

func TestStyledProgressReporter(t *testing.T) {
	var out bytes.Buffer
	reporter := NewStyledProgressReporter(&out, false)
	reporter.Report(app.ProgressEvent{Phase: "applying", Message: "Applying resource"})
	complete := app.ProgressEvent{Phase: "complete", Current: 1, Total: 1, Message: "Resource applied"}
	reporter.Report(complete)
	reporter.Report(complete)
	if !strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("output = %q, want ANSI escapes", out.String())
	}
	if got, want := stripANSI(out.String()), "applying Applying resource\n✓ Resource applied (1/1)\n✓ Resource applied (1/1)\n"; got != want {
		t.Fatalf("stripped output = %q, want %q", got, want)
	}
}

func TestStyledProgressReporterBar(t *testing.T) {
	bar := barOf(25) + "  50%"
	var out bytes.Buffer
	reporter := NewStyledProgressReporter(&out, false)
	reporter.Report(app.ProgressEvent{Phase: "working", Current: 10, Total: 20, Message: "Working"})
	reporter.Report(app.ProgressEvent{Phase: "working", Message: "Waiting"})
	reporter.Report(app.ProgressEvent{Phase: "complete", Current: 20, Total: 20, Message: "Done"})
	if !strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("output = %q, want ANSI escapes", out.String())
	}
	want := bar + " working Working (10/20)\nworking Waiting\n✓ Done (20/20)\n"
	if got := stripANSI(out.String()); got != want {
		t.Fatalf("stripped output = %q, want %q", got, want)
	}
	fills := map[string]bool{}
	for _, sequence := range ansiEscape.FindAllString(out.String(), -1) {
		if strings.HasPrefix(sequence, "\x1b[38;2;") {
			fills[sequence] = true
		}
	}
	if len(fills) < 2 {
		t.Fatalf("output = %q, want the filled cells blended across at least two colors", out.String())
	}
}

func TestSpinnerModelBar(t *testing.T) {
	newModel := func() spinnerModel {
		model := spinnerModel{spinner: spinner.New(), bar: newBar(), styles: newStyles()}
		model.bar.spring = harmonica.NewSpring(harmonica.FPS(barFPS), 200, 1) // settles in a few frames, so the test stays quick
		return model
	}

	t.Run("a counted label targets Current/Total and starts the glide", func(t *testing.T) {
		updated, cmd := newModel().Update(labelFor(app.ProgressEvent{Phase: "working", Current: 10, Total: 20, Message: "Working"}))
		model := updated.(spinnerModel)
		if got := model.bar.target; got != 0.5 || cmd == nil {
			t.Fatalf("target = %v, command = %v; want 0.5 and a frame command", got, cmd)
		}
		if got, want := stripANSI(model.View().Content), barOf(0)+"   0% working Working (10/20)"; got != want {
			t.Fatalf("view before any frame = %q, want %q (no jump)", got, want)
		}
		for range 200 {
			if !model.bar.ticking {
				break
			}
			next, frameCmd := model.Update(cmd())
			model, cmd = next.(spinnerModel), frameCmd
		}
		if got, want := stripANSI(model.View().Content), barOf(25)+"  50% working Working (10/20)"; got != want {
			t.Fatalf("view after the glide = %q, want %q", got, want)
		}
		if cmd != nil || model.bar.ticking {
			t.Fatalf("command = %v, ticking = %t; want the frames stopped once the glide settled", cmd, model.bar.ticking)
		}
	})

	t.Run("a second label mid-glide retargets without a second frame chain", func(t *testing.T) {
		updated, cmd := newModel().Update(labelFor(app.ProgressEvent{Phase: "working", Current: 10, Total: 20, Message: "Working"}))
		if cmd == nil {
			t.Fatal("first counted label returned no frame command")
		}
		updated, cmd = updated.Update(labelFor(app.ProgressEvent{Phase: "working", Current: 11, Total: 20, Message: "Working"}))
		if got := updated.(spinnerModel).bar.target; got != 0.55 || cmd != nil {
			t.Fatalf("target = %v, command = %v; want 0.55 and no new frame command", got, cmd)
		}
	})

	t.Run("an uncounted label shows the spinner and no bar", func(t *testing.T) {
		updated, cmd := newModel().Update(labelFor(app.ProgressEvent{Phase: "working", Message: "Waiting"}))
		if got := stripANSI(updated.View().Content); cmd != nil || strings.ContainsAny(got, "█░%") {
			t.Fatalf("view = %q, command = %v; want the spinner, no bar and no frame command", got, cmd)
		}
	})
}

func TestStyledProgressReporterSerializesConcurrentCalls(t *testing.T) {
	for _, animate := range []bool{false, true} {
		t.Run(fmt.Sprintf("animate=%t", animate), func(t *testing.T) {
			var out bytes.Buffer
			reporter := NewStyledProgressReporter(&out, animate)
			const calls = 100
			var workers sync.WaitGroup
			workers.Add(calls)
			for range calls {
				go func() {
					defer workers.Done()
					reporter.Report(app.ProgressEvent{Phase: "working", Message: "Update"})
				}()
			}
			within(t, workers.Wait)
			within(t, reporter.Close)
			if animate {
				return
			}
			if got := strings.Count(stripANSI(out.String()), "working Update\n"); got != calls {
				t.Fatalf("reported lines = %d, want %d", got, calls)
			}
		})
	}
}

func TestAnimatedProgressReporterRestoresTheTerminal(t *testing.T) {
	const hideCursor, showCursor = "\x1b[?25l", "\x1b[?25h"
	applying := app.ProgressEvent{Phase: "applying", Message: "Applying resource"}

	t.Run("completion leaves one check mark line after the cursor is restored", func(t *testing.T) {
		var out bytes.Buffer
		reporter := NewStyledProgressReporter(&out, true)
		reporter.Report(applying)
		reporter.Report(app.ProgressEvent{Phase: "complete", Current: 1, Total: 1, Message: "Resource applied"})
		within(t, reporter.Close)
		got := out.String()
		if strings.Count(got, "✓") != 1 || !strings.HasSuffix(stripANSI(got), "✓ Resource applied (1/1)\n") {
			t.Fatalf("output = %q, want one completion line last", got)
		}
		if shown, check := strings.LastIndex(got, showCursor), strings.Index(got, "✓"); shown < 0 || shown > check {
			t.Fatalf("output = %q, want the cursor restored before the completion line", got)
		}
	})

	t.Run("counted progress ends with one completion line after the cursor is restored", func(t *testing.T) {
		var out bytes.Buffer
		reporter := NewStyledProgressReporter(&out, true)
		for step := 1; step <= 3; step++ {
			reporter.Report(app.ProgressEvent{Phase: "working", Current: step, Total: 4, Message: "Working"})
		}
		reporter.Report(app.ProgressEvent{Phase: "complete", Current: 4, Total: 4, Message: "Done"})
		within(t, reporter.Close)
		got := out.String()
		if strings.Count(got, "✓") != 1 || !strings.HasSuffix(stripANSI(got), "✓ Done (4/4)\n") {
			t.Fatalf("output = %q, want one completion line last", got)
		}
		if shown, check := strings.LastIndex(got, showCursor), strings.Index(got, "✓"); shown < 0 || shown > check {
			t.Fatalf("output = %q, want the cursor restored before the completion line", got)
		}
	})

	t.Run("close without completion restores the cursor and prints no check mark", func(t *testing.T) {
		var out bytes.Buffer
		reporter := NewStyledProgressReporter(&out, true)
		reporter.Report(applying)
		within(t, reporter.Close)
		got := out.String()
		if strings.Contains(got, "✓") {
			t.Fatalf("output = %q, want no completion line", got)
		}
		if hidden, shown := strings.LastIndex(got, hideCursor), strings.LastIndex(got, showCursor); hidden < 0 || shown < hidden {
			t.Fatalf("output = %q, want the cursor hidden then restored", got)
		}
	})

	t.Run("close is idempotent, safe first, and silences later reports", func(t *testing.T) {
		var out bytes.Buffer
		reporter := NewStyledProgressReporter(&out, true)
		within(t, reporter.Close)
		within(t, reporter.Close)
		if out.Len() != 0 {
			t.Fatalf("output after Close without progress = %q, want none", out.String())
		}
		within(t, func() { reporter.Report(applying) })
		if out.Len() != 0 {
			t.Fatalf("output after Close and Report = %q, want none", out.String())
		}
	})

	t.Run("progress after an early completion prints a line instead of starting a spinner", func(t *testing.T) {
		var out bytes.Buffer
		reporter := NewStyledProgressReporter(&out, true)
		reporter.Report(app.ProgressEvent{Phase: "complete", Current: 1, Total: 1, Message: "Resource applied"})
		reporter.Report(applying)
		within(t, reporter.Close)
		if got := stripANSI(out.String()); got != "✓ Resource applied (1/1)\napplying Applying resource\n" || strings.Contains(out.String(), hideCursor) {
			t.Fatalf("output = %q, want the completion line then a plain styled line and no terminal control", out.String())
		}
	})

	t.Run("a completion event alone never starts the spinner", func(t *testing.T) {
		var out bytes.Buffer
		reporter := NewStyledProgressReporter(&out, true)
		reporter.Report(app.ProgressEvent{Phase: "complete", Current: 1, Total: 1, Message: "Resource applied"})
		within(t, reporter.Close)
		if got := stripANSI(out.String()); got != "✓ Resource applied (1/1)\n" || strings.Contains(out.String(), hideCursor) {
			t.Fatalf("output = %q, want only the completion line and no terminal control", out.String())
		}
	})
}

// within fails the test when fn does not return promptly, so a deadlock reads as a failure.
func within(t *testing.T, fn func()) {
	t.Helper()
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		fn()
	}()
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("call did not return within 5s")
	}
}

func TestNoopProgressReporter(t *testing.T) {
	NoopProgressReporter{}.Report(app.ProgressEvent{Phase: "complete", Message: "Done"})
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

// barOf is the bar text with filled of its barCells cells filled.
func barOf(filled int) string {
	return strings.Repeat("█", filled) + strings.Repeat("░", barCells-filled)
}

func TestBarMovesInEighthsOfACell(t *testing.T) {
	bar := newBar()
	for _, test := range []struct {
		name     string
		fraction float64
		want     string
	}{
		{name: "empty", fraction: 0, want: barOf(0) + "   0%"},
		{name: "a whole number of cells has no partial glyph", fraction: 0.5, want: barOf(25) + "  50%"},
		{name: "three eighths of the third cell", fraction: (2*8 + 3) / float64(barCells*8), want: "██▍" + strings.Repeat("░", barCells-3) + "   5%"},
		{name: "seven eighths", fraction: 7 / float64(barCells*8), want: "▉" + strings.Repeat("░", barCells-1) + "   2%"},
		{name: "full", fraction: 1, want: barOf(barCells) + " 100%"},
		{name: "out of range is clamped", fraction: 1.5, want: barOf(barCells) + " 100%"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := stripANSI(bar.viewAs(test.fraction)); got != test.want {
				t.Fatalf("viewAs(%v) = %q, want %q", test.fraction, got, test.want)
			}
		})
	}
}
