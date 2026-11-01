package output

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

func TestRendererText(t *testing.T) {
	resources := []domain.Resource{{Name: "example-1", Status: "ready"}, {Name: "example-2", Status: "pending"}}
	tests := []struct {
		name string
		run  func(*Renderer) error
		want string
	}{
		{
			name: "list",
			run:  func(renderer *Renderer) error { return renderer.RenderList(resources) },
			want: "NAME        STATUS\nexample-1   ready\nexample-2   pending\n",
		},
		{
			name: "resource",
			run:  func(renderer *Renderer) error { return renderer.RenderResource(resources[0]) },
			want: "example-1: ready\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			renderer := Renderer{Writer: &output, Format: FormatText}
			if err := test.run(&renderer); err != nil {
				t.Fatal(err)
			}
			if got := output.String(); got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRendererJSON(t *testing.T) {
	resources := []domain.Resource{{Name: "example-1", Status: "ready"}, {Name: "example-2", Status: "pending"}}
	tests := []struct {
		name  string
		run   func(*Renderer) error
		check func(*testing.T, []byte)
	}{
		{
			name: "list",
			run:  func(renderer *Renderer) error { return renderer.RenderList(resources) },
			check: func(t *testing.T, data []byte) {
				var got []map[string]string
				if err := json.Unmarshal(data, &got); err != nil {
					t.Fatal(err)
				}
				want := []map[string]string{{"name": "example-1", "status": "ready"}, {"name": "example-2", "status": "pending"}}
				if len(got) != len(want) {
					t.Fatalf("decoded list = %#v, want %#v", got, want)
				}
				for i := range want {
					if len(got[i]) != 2 || got[i]["name"] != want[i]["name"] || got[i]["status"] != want[i]["status"] {
						t.Fatalf("decoded item %d = %#v, want %#v", i, got[i], want[i])
					}
				}
			},
		},
		{
			name: "resource",
			run:  func(renderer *Renderer) error { return renderer.RenderResource(resources[0]) },
			check: func(t *testing.T, data []byte) {
				var got map[string]string
				if err := json.Unmarshal(data, &got); err != nil {
					t.Fatal(err)
				}
				if len(got) != 2 || got["name"] != "example-1" || got["status"] != "ready" {
					t.Fatalf("decoded resource = %#v", got)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var first, second bytes.Buffer
			if err := (Renderer{Writer: &first, Format: FormatJSON}).run(test.run); err != nil {
				t.Fatal(err)
			}
			if err := (Renderer{Writer: &second, Format: FormatJSON}).run(test.run); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(first.Bytes(), second.Bytes()) {
				t.Fatalf("repeated JSON differs: %q vs %q", first.String(), second.String())
			}
			test.check(t, first.Bytes())
		})
	}
}

func TestDetectTerminalCapabilities(t *testing.T) {
	terminal := terminalWriter{info: testFileInfo{mode: os.ModeCharDevice}}
	pipe := terminalWriter{info: testFileInfo{mode: 0}}
	got := DetectTerminalCapabilities(terminal, pipe)
	if !got.Stdout || got.Stderr {
		t.Fatalf("capabilities = %+v, want stdout terminal and stderr non-terminal", got)
	}
}

func TestColorEnabled(t *testing.T) {
	tests := []struct {
		name      string
		format    Format
		mode      ColorMode
		terminals TerminalCapabilities
		noColor   bool
		want      bool
	}{
		{name: "auto on stdout terminal", format: FormatText, mode: ColorAuto, terminals: TerminalCapabilities{Stdout: true}, want: true},
		{name: "stderr terminal does not enable stdout color", format: FormatText, mode: ColorAuto, terminals: TerminalCapabilities{Stderr: true}},
		{name: "no color disables auto", format: FormatText, mode: ColorAuto, terminals: TerminalCapabilities{Stdout: true}, noColor: true},
		{name: "always overrides no color and tty", format: FormatText, mode: ColorAlways, noColor: true, want: true},
		{name: "never overrides stdout tty", format: FormatText, mode: ColorNever, terminals: TerminalCapabilities{Stdout: true}},
		{name: "json ignores always", format: FormatJSON, mode: ColorAlways, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ColorEnabled(test.format, test.mode, test.terminals, test.noColor); got != test.want {
				t.Fatalf("ColorEnabled() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestRendererNeverAddsANSI(t *testing.T) {
	for _, format := range []Format{FormatText, FormatJSON} {
		t.Run(string(format), func(t *testing.T) {
			var output bytes.Buffer
			renderer := Renderer{Writer: &output, Format: format}
			if err := renderer.RenderList([]domain.Resource{{Name: "example-1", Status: "ready"}}); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(output.String(), "\x1b[") {
				t.Fatalf("output contains ANSI escape: %q", output.String())
			}
		})
	}
}

type terminalWriter struct {
	info os.FileInfo
	err  error
}

func (w terminalWriter) Write(data []byte) (int, error) { return len(data), nil }

func (w terminalWriter) Stat() (os.FileInfo, error) { return w.info, w.err }

type testFileInfo struct{ mode os.FileMode }

func (i testFileInfo) Name() string       { return "test" }
func (i testFileInfo) Size() int64        { return 0 }
func (i testFileInfo) Mode() os.FileMode  { return i.mode }
func (i testFileInfo) ModTime() time.Time { return time.Time{} }
func (i testFileInfo) IsDir() bool        { return false }
func (i testFileInfo) Sys() any           { return nil }

func (r Renderer) run(render func(*Renderer) error) error { return render(&r) }
