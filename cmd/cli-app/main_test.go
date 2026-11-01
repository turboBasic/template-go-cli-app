package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
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

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }
