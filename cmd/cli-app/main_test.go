package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/cli"
	"github.com/turboBasic/template-go-cli-app/internal/config"
	"github.com/turboBasic/template-go-cli-app/internal/domain"
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
	command := cli.NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(reader, logger), nil, nil
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

func TestSignalHelperProcess(t *testing.T) {
	if os.Getenv("CLI_APP_SIGNAL_HELPER") != "1" {
		return
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "no-user-config"))
	configFile := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(configFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	resources := memory.NewResourceReader()
	reader := signalBlockingReader{ResourceReader: resources}
	command := cli.NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(reader, logger), func(ctx context.Context) error {
			_, _ = fmt.Fprintln(os.Stdout, "cleanup started")
			if os.Getenv("CLI_APP_SIGNAL_HELPER_BLOCK_CLEANUP") == "1" {
				<-ctx.Done()
				return ctx.Err()
			}
			return nil
		}, nil
	}, cli.BuildInfo{}, resources.Names())
	command.SetArgs([]string{"--config", configFile, "resource", "list"})
	code := executeWithSignals(func(ctx context.Context) error {
		_, _ = fmt.Fprintln(os.Stdout, "ready")
		command.SetContext(ctx)
		return runCommand(command)
	}, os.Stderr)
	exitWithCode(code)
}

func TestFirstSignalCancelsCleansUpAndMapsExitCode(t *testing.T) {
	for _, test := range []struct {
		name     string
		signal   os.Signal
		exitCode int
	}{
		{name: "SIGINT", signal: os.Interrupt, exitCode: 130},
		{name: "SIGTERM", signal: syscall.SIGTERM, exitCode: 143},
	} {
		t.Run(test.name, func(t *testing.T) {
			command, lines := startSignalHelper(t, false)
			if got := nextSignalLine(t, lines); got != "ready" {
				t.Fatalf("helper output = %q, want ready", got)
			}
			if got := nextSignalLine(t, lines); got != "operation started" {
				t.Fatalf("helper output = %q, want operation started", got)
			}
			if err := command.Process.Signal(test.signal); err != nil {
				t.Fatal(err)
			}
			if got := nextSignalLine(t, lines); got != "cleanup started" {
				t.Fatalf("helper output = %q, want cleanup started", got)
			}
			if got := waitSignalExitCode(t, command); got != test.exitCode {
				t.Fatalf("signal helper exit code = %d, want %d", got, test.exitCode)
			}
		})
	}
}

func TestSecondSignalBypassesCleanup(t *testing.T) {
	command, lines := startSignalHelper(t, true)
	if got := nextSignalLine(t, lines); got != "ready" {
		t.Fatalf("helper output = %q, want ready", got)
	}
	if got := nextSignalLine(t, lines); got != "operation started" {
		t.Fatalf("helper output = %q, want operation started", got)
	}
	if err := command.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	if got := nextSignalLine(t, lines); got != "cleanup started" {
		t.Fatalf("helper output = %q, want cleanup started", got)
	}
	started := time.Now()
	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if got := waitSignalExitCode(t, command); got != 143 {
		t.Fatalf("signal helper exit code = %d, want 143", got)
	}
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Fatalf("second signal took %s to bypass 5s cleanup", elapsed)
	}
}

func startSignalHelper(t *testing.T, blockCleanup bool) (*exec.Cmd, <-chan string) {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run=^TestSignalHelperProcess$")
	blockValue := "0"
	if blockCleanup {
		blockValue = "1"
	}
	command.Env = append(os.Environ(), "CLI_APP_SIGNAL_HELPER=1", "CLI_APP_SIGNAL_HELPER_BLOCK_CLEANUP="+blockValue)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	lines := make(chan string)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
	return command, lines
}

func nextSignalLine(t *testing.T, lines <-chan string) string {
	t.Helper()
	select {
	case line, ok := <-lines:
		if !ok {
			t.Fatal("signal helper exited before writing expected output")
		}
		return line
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for signal helper output")
		return ""
	}
}

func waitSignalExitCode(t *testing.T, command *exec.Cmd) int {
	t.Helper()
	err := command.Wait()
	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("signal helper process: %v", err)
	}
	return exitError.ExitCode()
}

type signalBlockingReader struct{ app.ResourceReader }

func (r signalBlockingReader) List(ctx context.Context) ([]domain.Resource, error) {
	_, _ = fmt.Fprintln(os.Stdout, "operation started")
	<-ctx.Done()
	return nil, ctx.Err()
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
