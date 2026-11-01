package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFullPrecedence(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user.toml")
	project := filepath.Join(root, "project.toml")
	writeFile(t, user, "log-level = 'info'\noutput = 'text'\n")
	writeFile(t, project, "log-level = 'error'\n")

	got, err := load(ConfigFiles{User: user, Project: project}, []string{
		"CLI_APP_LOG_LEVEL=warn",
		"CLI_APP_QUIET=true",
		"CLI_APP_UNKNOWN=ignored",
	}, map[string]FlagValue{
		"log-level": {Value: "debug", Changed: true, Source: "--log-level"},
		"output":    {Value: "json", Changed: true, Source: "--output"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := Config{LogLevel: "debug", Output: "json", Color: "auto", Quiet: true}
	if got.Config != want {
		t.Fatalf("Config = %+v, want %+v", got.Config, want)
	}
	for key, wantSource := range map[string]Source{
		"log-level": "--log-level",
		"output":    "--output",
		"quiet":     "CLI_APP_QUIET",
		"color":     defaultSource,
	} {
		if got.Sources[key] != wantSource {
			t.Errorf("source for %q = %q, want %q", key, got.Sources[key], wantSource)
		}
	}
}

func TestLoadUnchangedFlagLeavesEnvironmentEffective(t *testing.T) {
	got, err := load(ConfigFiles{}, []string{"CLI_APP_LOG_LEVEL=info"}, map[string]FlagValue{
		"log-level": {Value: "warn", Changed: false, Source: "--log-level"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Config.LogLevel != "info" || got.Sources["log-level"] != "CLI_APP_LOG_LEVEL" {
		t.Fatalf("effective log level/source = %q/%q, want info/CLI_APP_LOG_LEVEL", got.Config.LogLevel, got.Sources["log-level"])
	}
}

func TestLoadInvalidEnvironmentTypeNamesVariable(t *testing.T) {
	_, err := load(ConfigFiles{}, []string{"CLI_APP_QUIET=sometimes"}, nil)
	if err == nil || !strings.Contains(err.Error(), "CLI_APP_QUIET") || !strings.Contains(err.Error(), "invalid boolean") {
		t.Fatalf("load() error = %v, want invalid CLI_APP_QUIET error", err)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
