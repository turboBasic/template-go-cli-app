package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
	"github.com/turboBasic/template-go-cli-app/internal/output"
)

type configRow struct {
	Key    string `json:"key"`
	Source string `json:"source"`
	Origin string `json:"origin"`
}

func configShowFixture(t *testing.T) (user, project string, newCommand func(args ...string) (*cobra.Command, *bytes.Buffer)) {
	t.Helper()
	root := t.TempDir()
	user = filepath.Join(root, "xdg", "cli-app", "config.toml")
	project = filepath.Join(root, "project.toml")
	for path, data := range map[string]string{
		user:    "[log.file]\nlevel = 'info'\n[log.console]\nlevel = 'warn'\n",
		project: "[log.console]\nlevel = 'error'\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg"))
	t.Setenv("CLI_APP_COLOR", "never")
	return user, project, func(args ...string) (*cobra.Command, *bytes.Buffer) {
		reader := memory.NewResourceReader()
		command := NewRootCommand(reader, testBuildInfo(), reader.Names())
		var out bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&bytes.Buffer{})
		command.SetArgs(append([]string{"--config", project, "--quiet"}, args...))
		return command, &out
	}
}

func TestConfigShowReportsEverySource(t *testing.T) {
	user, project, newCommand := configShowFixture(t)
	want := map[string]configRow{
		"log.file.level":    {Source: "user file", Origin: user},
		"log.console.level": {Source: "project file", Origin: project},
		"color":             {Source: "environment variable", Origin: "CLI_APP_COLOR"},
		"quiet":             {Source: "flag", Origin: "--quiet"},
		"output":            {Source: "default"},
	}

	command, out := newCommand("--output", "json", "config", "show")
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	var rows []configRow
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, row := range rows {
		if row.Key == "output" {
			continue // --output json makes it a flag here; the default row is checked in the text run
		}
		if w, ok := want[row.Key]; ok {
			found++
			if row.Source != w.Source || row.Origin != w.Origin {
				t.Errorf("%s = (%q, %q), want (%q, %q)", row.Key, row.Source, row.Origin, w.Source, w.Origin)
			}
		}
	}
	if found != len(want)-1 {
		t.Errorf("found %d of %d expected JSON rows", found, len(want)-1)
	}

	command, out = newCommand("config", "show")
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, line := range []string{
		"user file (" + user + ")",
		"project file (" + project + ")",
		"environment variable (CLI_APP_COLOR)",
		"flag (--quiet)",
	} {
		if !strings.Contains(text, line) {
			t.Errorf("text output missing %q:\n%s", line, text)
		}
	}
	if !regexp.MustCompile(`log\.file\.level\s+"info"`).MatchString(text) {
		t.Errorf("text output missing the log.file.level row:\n%s", text)
	}
	sawOutput := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "output ") {
			sawOutput = true
			if !strings.HasSuffix(line, "default") {
				t.Errorf("output row = %q, want source default", line)
			}
		}
	}
	if !sawOutput {
		t.Errorf("text output has no output row:\n%s", text)
	}
}

func TestConfigShowKeyFilter(t *testing.T) {
	_, _, newCommand := configShowFixture(t)
	keys := func(args ...string) []string {
		t.Helper()
		command, out := newCommand(append([]string{"--output", "json", "config", "show"}, args...)...)
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		var rows []configRow
		if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
			t.Fatal(err)
		}
		var result []string
		for _, row := range rows {
			result = append(result, row.Key)
		}
		return result
	}
	if got := strings.Join(keys("log"), ","); got != "log.console.level,log.file.level,log.file.path" {
		t.Errorf("config show log = %s", got)
	}
	if got := keys("log.file.level"); len(got) != 1 || got[0] != "log.file.level" {
		t.Errorf("config show log.file.level = %v", got)
	}

	command, out := newCommand("config", "show", "lo")
	if err := command.Execute(); !errors.Is(err, app.ErrValidation) || out.Len() != 0 {
		t.Errorf("config show lo = %v with stdout %q, want validation error and empty stdout", err, out.String())
	}
	command, _ = newCommand("config", "nope")
	if err := command.Execute(); !errors.Is(err, app.ErrValidation) {
		t.Errorf("config nope = %v, want validation error", err)
	}
}

func TestConfigShowMasksSensitiveValues(t *testing.T) {
	entries := []config.Entry{
		{Key: "output", Value: "text", Source: config.KindDefault},
		{Key: "provider.api_token", Value: "secret-value", Source: config.KindUserFile, Origin: "/u/config.toml"},
	}
	for _, format := range []output.Format{output.FormatText, output.FormatJSON} {
		t.Run(string(format), func(t *testing.T) {
			var out bytes.Buffer
			renderer := output.Renderer{Writer: &out, Options: output.Options{Format: format, Color: output.ColorNever}}
			command := newConfigCommand(func() []config.Entry { return entries }, func() output.Renderer { return renderer }, func(context.Context) error { return nil })
			command.SetArgs([]string{"show"})
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			got := out.String()
			if strings.Contains(got, "secret-value") || !strings.Contains(got, "[REDACTED]") || !strings.Contains(got, "text") {
				t.Errorf("output = %q, want masked token and the real output value", got)
			}
		})
	}
}

func TestConfigShowCredentialReferenceIsNotMasked(t *testing.T) {
	_, _, newCommand := configShowFixture(t)
	t.Setenv("CLI_APP_RESOURCE_CREDENTIAL", "op://vault/item/field")
	command, out := newCommand("--output", "json", "config", "show", "resource.credential")
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	var rows []configRow
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Key != "resource.credential" || rows[0].Source != "environment variable" {
		t.Fatalf("rows = %+v, want one environment row", rows)
	}
	if got := out.String(); !strings.Contains(got, "op://vault/item/field") || strings.Contains(got, "[REDACTED]") {
		t.Fatalf("output = %q, want the unmasked reference", got)
	}
}
