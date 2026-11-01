package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

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
			renderer := Renderer{Writer: &output, Options: Options{Format: FormatText}}
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
			if err := (Renderer{Writer: &first, Options: Options{Format: FormatJSON}}).run(test.run); err != nil {
				t.Fatal(err)
			}
			if err := (Renderer{Writer: &second, Options: Options{Format: FormatJSON}}).run(test.run); err != nil {
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
		t.Skip("null device is not reported as a character device on this platform")
	}
	if isTerminal(file) {
		t.Fatal("null device was detected as a terminal")
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
			renderer := Renderer{Writer: &output, Options: Options{Format: format}}
			if err := renderer.RenderList([]domain.Resource{{Name: "example-1", Status: "ready"}}); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(output.String(), "\x1b[") {
				t.Fatalf("output contains ANSI escape: %q", output.String())
			}
		})
	}
}

func (r Renderer) run(render func(*Renderer) error) error { return render(&r) }
