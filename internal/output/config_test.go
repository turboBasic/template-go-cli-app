package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/config"
)

var configFixture = []config.Entry{
	{Key: "output", Value: "text", Source: config.KindDefault},
	{Key: "provider.api_token", Value: "[REDACTED]", Source: config.KindUserFile, Origin: "/u/config.toml"},
	{Key: "quiet", Value: true, Source: config.KindFlag, Origin: "--quiet"},
}

func renderConfig(t *testing.T, options Options) string {
	t.Helper()
	var out bytes.Buffer
	if err := (Renderer{Writer: &out, Options: options}).RenderConfig(configFixture); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestRenderConfigText(t *testing.T) {
	want := "KEY                 VALUE         SOURCE\n" +
		"output              \"text\"        default\n" +
		"provider.api_token  \"[REDACTED]\"  user file (/u/config.toml)\n" +
		"quiet               true          flag (--quiet)\n"
	if got := renderConfig(t, Options{Format: FormatText, Color: ColorNever}); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRenderConfigTextColorStylesOnlyTheHeader(t *testing.T) {
	plain := renderConfig(t, Options{Format: FormatText, Color: ColorNever})
	styled := renderConfig(t, Options{Format: FormatText, Color: ColorAlways})
	lines := strings.Split(styled, "\n")
	if !strings.Contains(lines[0], "\x1b[") {
		t.Fatalf("header = %q, want styled", lines[0])
	}
	if rest := strings.Join(lines[1:], "\n"); strings.Contains(rest, "\x1b") {
		t.Fatalf("body = %q, want plain", rest)
	}
	if stripANSI(styled) != plain {
		t.Fatalf("stripped = %q, want %q", stripANSI(styled), plain)
	}
}

func TestRenderConfigJSON(t *testing.T) {
	got := renderConfig(t, Options{Format: FormatJSON, Color: ColorAlways})
	if strings.Contains(got, "\x1b") {
		t.Fatalf("output = %q, want no escapes", got)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(got), &items); err != nil {
		t.Fatal(err)
	}
	if _, ok := items[0]["origin"]; ok {
		t.Errorf("default row = %v, want no origin", items[0])
	}
	if items[2]["value"] != true || items[2]["origin"] != "--quiet" || items[2]["source"] != "flag" {
		t.Errorf("quiet row = %v, want boolean true from flag --quiet", items[2])
	}
}
