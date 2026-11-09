package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"regexp"
	"testing"
)

type applicationResult struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiEscape.ReplaceAllString(s, "") }

func TestRenderResultTextCallbackReceivesWriterAndOptions(t *testing.T) {
	var got bytes.Buffer
	options := Options{Format: FormatText, Color: ColorAlways}
	called := false
	err := (Renderer{Writer: &got, Options: options}).RenderResult(applicationResult{ID: 7}, func(w io.Writer, gotOptions Options) error {
		called = true
		if gotOptions != options {
			t.Fatalf("options = %+v, want %+v", gotOptions, options)
		}
		_, err := io.WriteString(w, "application text\n")
		return err
	})
	if err != nil || !called || got.String() != "application text\n" {
		t.Fatalf("RenderResult() = %v, called = %t, output = %q", err, called, got.String())
	}
}

func TestRenderResultJSONSkipsTextCallback(t *testing.T) {
	var got bytes.Buffer
	called := false
	err := (Renderer{Writer: &got, Options: Options{Format: FormatJSON}}).RenderResult(applicationResult{ID: 7, Status: "ready"}, func(io.Writer, Options) error {
		called = true
		return errors.New("must not be called")
	})
	if err != nil || called || got.String() != "{\"id\":7,\"status\":\"ready\"}\n" {
		t.Fatalf("RenderResult() = %v, called = %t, output = %q", err, called, got.String())
	}
}

type resultFailingWriter struct{ err error }

func (w resultFailingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestRenderResultReturnsWriterAndEncodingErrors(t *testing.T) {
	wantErr := errors.New("injected write failure")
	for _, test := range []struct {
		name   string
		format Format
		value  any
		text   func(io.Writer, Options) error
	}{
		{name: "text", format: FormatText, text: func(w io.Writer, _ Options) error { _, err := io.WriteString(w, "text"); return err }},
		{name: "json", format: FormatJSON, value: applicationResult{ID: 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := (Renderer{Writer: resultFailingWriter{err: wantErr}, Options: Options{Format: test.format}}).RenderResult(test.value, test.text)
			if !errors.Is(err, wantErr) {
				t.Fatalf("RenderResult() error = %v, want writer error", err)
			}
		})
	}
	var invalid bytes.Buffer
	if err := (Renderer{Writer: &invalid, Options: Options{Format: FormatJSON}}).RenderResult(map[string]any{"bad": mathInf()}, nil); err == nil {
		t.Fatal("RenderResult() error = nil, want JSON encoding error")
	}
}

func TestDetectTerminalCapabilities(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	got := detectTerminalCapabilities(stdout, stderr, func(writer io.Writer) bool { return writer == stdout })
	if !got.Stdout || got.Stderr {
		t.Fatalf("capabilities = %+v, want injected stdout terminal and stderr non-terminal", got)
	}
}

func TestNonTerminalCharacterDevice(t *testing.T) {
	file, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("close null device: %v", err)
		}
	}()
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		t.Skip("/dev/null is not a character device")
	}
	if isTerminal(file) {
		t.Fatal("isTerminal(/dev/null) = true, want false")
	}
}

func TestRenderResultPreservesJSONNewline(t *testing.T) {
	var out bytes.Buffer
	if err := (Renderer{Writer: &out, Options: Options{Format: FormatJSON}}).RenderResult(applicationResult{ID: 1}, nil); err != nil {
		t.Fatal(err)
	}
	var decoded applicationResult
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON = %q: %v", out.String(), err)
	}
	if decoded.ID != 1 || out.Bytes()[out.Len()-1] != '\n' {
		t.Fatalf("JSON = %q, want encoded result with newline", out.String())
	}
}

func mathInf() float64 { return math.Inf(1) }
