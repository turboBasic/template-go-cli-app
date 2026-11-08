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
		{name: "always overrides NO_COLOR", options: Options{Format: FormatText, Color: ColorAlways, NoColor: true}, wantStyled: true},
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

func TestRendererListColor(t *testing.T) {
	tests := []struct {
		name      string
		resources []domain.Resource
		contains  []string
	}{
		{
			name:      "fixture",
			resources: []domain.Resource{{Name: "example-1", Status: "ready"}, {Name: "example-2", Status: "pending"}},
			contains:  []string{"\x1b[33m"},
		},
		{
			name:      "name wider than the column",
			resources: []domain.Resource{{Name: "a-fourteen-char", Status: "ready"}, {Name: "b", Status: "unknown"}},
			contains:  []string{" unknown\n"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var plain, styled bytes.Buffer
			if err := (Renderer{Writer: &plain, Options: Options{Format: FormatText}}).RenderList(test.resources); err != nil {
				t.Fatal(err)
			}
			if err := (Renderer{Writer: &styled, Options: Options{Format: FormatText, Color: ColorAlways}}).RenderList(test.resources); err != nil {
				t.Fatal(err)
			}
			got := styled.String()
			if !strings.Contains(got, "\x1b[1m") || !strings.Contains(got, "\x1b[32m") {
				t.Fatalf("output = %q, want bold header and green ready", got)
			}
			for _, fragment := range test.contains {
				if !strings.Contains(got, fragment) {
					t.Fatalf("output = %q, want fragment %q", got, fragment)
				}
			}
			if stripped := stripANSI(got); stripped != plain.String() {
				t.Fatalf("stripped = %q, want plain %q", stripped, plain.String())
			}
		})
	}
}
