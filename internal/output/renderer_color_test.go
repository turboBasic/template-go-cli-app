package output

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiEscape.ReplaceAllString(s, "") }

func TestRendererResourceColor(t *testing.T) {
	resource := domain.Resource{Name: "example-1", Status: "ready"}
	tests := []struct {
		name       string
		options    Options
		wantStyled bool
	}{
		{name: "always", options: Options{Format: FormatText, Color: ColorAlways}, wantStyled: true},
		{name: "auto on stdout terminal", options: Options{Format: FormatText, Color: ColorAuto, Terminals: TerminalCapabilities{Stdout: true}}, wantStyled: true},
		{name: "auto redirected", options: Options{Format: FormatText, Color: ColorAuto}},
		{name: "auto with NO_COLOR", options: Options{Format: FormatText, Color: ColorAuto, Terminals: TerminalCapabilities{Stdout: true}, NoColor: true}},
		{name: "never", options: Options{Format: FormatText, Color: ColorNever, Terminals: TerminalCapabilities{Stdout: true}}},
		{name: "json ignores always", options: Options{Format: FormatJSON, Color: ColorAlways}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			renderer := Renderer{Writer: &output, Options: test.options}
			if err := renderer.RenderResource(resource); err != nil {
				t.Fatal(err)
			}
			got := output.String()
			if styled := strings.Contains(got, "\x1b"); styled != test.wantStyled {
				t.Fatalf("output = %q, styled = %t, want %t", got, styled, test.wantStyled)
			}
			if test.wantStyled {
				if !strings.Contains(got, "\x1b[32m") {
					t.Fatalf("output = %q, want green status", got)
				}
				if plain := stripANSI(got); plain != "example-1: ready\n" {
					t.Fatalf("stripped output = %q, want plain text", plain)
				}
			}
		})
	}
}
