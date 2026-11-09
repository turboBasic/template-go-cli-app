package output

import (
	"bytes"
	"io"
	"testing"
)

func TestRendererColorEnabled(t *testing.T) {
	tests := []struct {
		name    string
		options Options
		want    bool
	}{
		{name: "always", options: Options{Format: FormatText, Color: ColorAlways}, want: true},
		{name: "always overrides NO_COLOR", options: Options{Format: FormatText, Color: ColorAlways, NoColor: true}, want: true},
		{name: "auto terminal", options: Options{Format: FormatText, Color: ColorAuto, Terminals: TerminalCapabilities{Stdout: true}}, want: true},
		{name: "auto redirected", options: Options{Format: FormatText, Color: ColorAuto}},
		{name: "NO_COLOR", options: Options{Format: FormatText, Color: ColorAuto, Terminals: TerminalCapabilities{Stdout: true}, NoColor: true}},
		{name: "never", options: Options{Format: FormatText, Color: ColorNever, Terminals: TerminalCapabilities{Stdout: true}}},
		{name: "json ignores always", options: Options{Format: FormatJSON, Color: ColorAlways}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := (Renderer{Options: test.options}).ColorEnabled(); got != test.want {
				t.Fatalf("ColorEnabled() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestRenderResultPassesColorPolicyToTextCallback(t *testing.T) {
	var got bytes.Buffer
	err := (Renderer{Writer: &got, Options: Options{Format: FormatText, Color: ColorAlways}}).RenderResult(struct{}{}, func(w io.Writer, options Options) error {
		if !ColorEnabled(options.Format, options.Color, options.Terminals.Stdout, options.NoColor) {
			t.Fatal("text callback did not receive enabled color policy")
		}
		_, err := io.WriteString(w, "ready\n")
		return err
	})
	if err != nil || got.String() != "ready\n" {
		t.Fatalf("RenderResult() = %v, output = %q", err, got.String())
	}
}
