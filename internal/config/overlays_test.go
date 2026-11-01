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
	writeFile(t, user, "output = 'text'\n[log.console]\nlevel = 'info'\n")
	writeFile(t, project, "[log.console]\nlevel = 'error'\n")

	got, err := load(ConfigFiles{User: user, Project: project}, []string{
		"CLI_APP_LOG_CONSOLE_LEVEL=warn",
		"CLI_APP_QUIET=true",
		"CLI_APP_UNKNOWN=ignored",
	}, map[string]FlagValue{
		"log-level": {Value: "debug", Changed: true, Source: "--log-level"},
		"output":    {Value: "json", Changed: true, Source: "--output"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := Config{LogLevel: "debug", Log: LogConfig{Console: ConsoleLogConfig{Level: "debug"}, File: FileLogConfig{Level: "debug"}}, Output: "json", Color: "auto", Quiet: true}
	if got.Config != want {
		t.Fatalf("Config = %+v, want %+v", got.Config, want)
	}
	for key, wantSource := range map[string]Source{
		"log.console.level": "--log-level",
		"output":            "--output",
		"quiet":             "CLI_APP_QUIET",
		"color":             defaultSource,
	} {
		if got.Sources[key] != wantSource {
			t.Errorf("source for %q = %q, want %q", key, got.Sources[key], wantSource)
		}
	}
}

func TestLoadUnchangedFlagLeavesEnvironmentEffective(t *testing.T) {
	got, err := load(ConfigFiles{}, []string{"CLI_APP_LOG_CONSOLE_LEVEL=info"}, map[string]FlagValue{
		"log-level": {Value: "warn", Changed: false, Source: "--log-level"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Config.Log.Console.Level != "info" || got.Sources["log.console.level"] != "CLI_APP_LOG_CONSOLE_LEVEL" {
		t.Fatalf("effective console log level/source = %q/%q, want info/CLI_APP_LOG_CONSOLE_LEVEL", got.Config.Log.Console.Level, got.Sources["log.console.level"])
	}
}

func TestLoadInvalidEnvironmentTypeNamesVariable(t *testing.T) {
	_, err := load(ConfigFiles{}, []string{"CLI_APP_QUIET=sometimes"}, nil)
	if err == nil || !strings.Contains(err.Error(), "CLI_APP_QUIET") || !strings.Contains(err.Error(), "invalid boolean") {
		t.Fatalf("load() error = %v, want invalid CLI_APP_QUIET error", err)
	}
}

func TestLoadNestedLoggingEnvironment(t *testing.T) {
	got, err := load(ConfigFiles{}, []string{
		"CLI_APP_LOG_CONSOLE_LEVEL=error",
		"CLI_APP_LOG_FILE_LEVEL=info",
		"CLI_APP_LOG_FILE_PATH=out/log.jsonl",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Config.Log.Console.Level != "error" || got.Config.Log.File.Level != "info" {
		t.Fatalf("logging levels = %q/%q, want error/info", got.Config.Log.Console.Level, got.Config.Log.File.Level)
	}
	if filepath.Base(got.Config.Log.File.Path) != "log.jsonl" || got.Sources["log.file.path"] != "CLI_APP_LOG_FILE_PATH" {
		t.Fatalf("file path/source = %q/%q, want working-directory path from CLI_APP_LOG_FILE_PATH", got.Config.Log.File.Path, got.Sources["log.file.path"])
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
