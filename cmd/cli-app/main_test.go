package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/cli"
	"github.com/turboBasic/template-go-cli-app/internal/config"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
)

func TestBuildInfoIncludesEmbeddedVCSMetadata(t *testing.T) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Skip("Go build metadata is unavailable")
	}
	var revision, buildDate string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.time":
			buildDate = setting.Value
		}
	}
	got := buildInfo()
	if got.Version != version {
		t.Errorf("Version = %q, want %q", got.Version, version)
	}
	if revision != "" && got.Commit != revision {
		t.Errorf("Commit = %q, want %q", got.Commit, revision)
	}
	if buildDate != "" && got.BuildDate != buildDate {
		t.Errorf("BuildDate = %q, want %q", got.BuildDate, buildDate)
	}
}

func TestExecuteReportsOutputFailureToStderrAndReturnsFailureCode(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("CLI_APP_LOG_LEVEL", "warn")
	t.Setenv("CLI_APP_OUTPUT", "text")
	t.Setenv("CLI_APP_COLOR", "never")
	t.Setenv("CLI_APP_QUIET", "false")
	configPath := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(configPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	reader := memory.NewResourceReader()
	command := cli.NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger) (*app.ResourceService, error) {
		return app.NewResourceService(reader, logger), nil
	}, cli.BuildInfo{}, reader.Names())
	command.SetArgs([]string{"--config", configPath, "--output", "json", "resource", "list"})
	command.SetOut(failingWriter{err: errors.New("injected output failure")})
	var stderr bytes.Buffer

	if code := execute(command.Execute, &stderr); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "injected output failure") {
		t.Fatalf("stderr = %q, want output failure", stderr.String())
	}
}

func TestExitCode(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want int
	}{
		{name: "validation", err: app.Classify(app.ErrValidation, errors.New("invalid input")), want: 2},
		{name: "timeout category", err: app.Classify(app.ErrTimeout, errors.New("operation timed out")), want: 124},
		{name: "deadline", err: context.DeadlineExceeded, want: 124},
		{name: "interrupt", err: fmt.Errorf("command: %w", signalCause(os.Interrupt)), want: 130},
		{name: "termination", err: fmt.Errorf("command: %w", signalCause(syscall.SIGTERM)), want: 143},
		{name: "other", err: errors.New("operation failed"), want: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := exitCode(test.err); got != test.want {
				t.Fatalf("exitCode(%v) = %d, want %d", test.err, got, test.want)
			}
		})
	}
}

func TestUnknownCommandUsesUsageExitCode(t *testing.T) {
	reader := memory.NewResourceReader()
	command := cli.NewRootCommand(reader, cli.BuildInfo{}, reader.Names())
	command.SetArgs([]string{"missing"})
	var stderr bytes.Buffer
	if got := execute(func() error { return runCommand(command) }, &stderr); got != 2 {
		t.Fatalf("exit code = %d, want 2; stderr = %q", got, stderr.String())
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }
