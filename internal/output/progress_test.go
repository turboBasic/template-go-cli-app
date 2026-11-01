package output

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"

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
			name: "consecutive identical completion is deduplicated",
			events: []app.ProgressEvent{
				{Phase: "applying", Current: 2, Total: 2, Message: "Applied"},
				{Phase: "applying", Current: 2, Total: 2, Message: "Applied"},
			},
			want: "applying: Applied (2/2)\n",
		},
		{
			name: "completion phase is retained after other progress",
			events: []app.ProgressEvent{
				{Phase: "applying", Message: "Applying"},
				{Phase: "complete", Message: "Done"},
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

func TestNoopProgressReporter(t *testing.T) {
	NoopProgressReporter{}.Report(app.ProgressEvent{Phase: "complete", Message: "Done"})
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
