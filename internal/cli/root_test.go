package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
	"github.com/turboBasic/template-go-cli-app/internal/domain"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
)

func TestResourceListCommand(t *testing.T) {
	command := newTestRootCommand(t)
	command.SetArgs([]string{"resource", "list"})
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := "NAME        STATUS\nexample-1   ready\nexample-2   pending\n"
	if got := output.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRootFeatures(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"help", "resource", "status"}, {"--version"}, {"completion", "bash"}, {"completion", "zsh"}, {"completion", "fish"}, {"completion", "powershell"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			command := newTestRootCommand(t)
			command.SetArgs(args)
			var output bytes.Buffer
			command.SetOut(&output)
			if err := command.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if output.Len() == 0 {
				t.Fatal("command output is empty")
			}
			if args[0] == "--version" && (!strings.Contains(output.String(), "1.2.3") || !strings.Contains(output.String(), "abc123") || !strings.Contains(output.String(), "test-build")) {
				t.Fatalf("version output = %q", output.String())
			}
		})
	}
}

func TestResourceStatusAndApplyCommands(t *testing.T) {
	for _, tc := range []struct {
		args string
		want string
	}{
		{args: "status example-2", want: "example-2: pending\n"},
		{args: "apply example-2", want: "example-2: ready\n"},
	} {
		t.Run(tc.args, func(t *testing.T) {
			command := newTestRootCommand(t)
			command.SetArgs(strings.Fields("resource " + tc.args))
			command.SetIn(bytes.NewBuffer(nil))
			var output bytes.Buffer
			command.SetOut(&output)
			if err := command.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if output.String() != tc.want {
				t.Fatalf("output = %q, want %q", output.String(), tc.want)
			}
		})
	}
}

func TestResourceCommandsJSONOutput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "list", args: []string{"resource", "list"}, want: "[{\"name\":\"example-1\",\"status\":\"ready\"},{\"name\":\"example-2\",\"status\":\"pending\"}]\n"},
		{name: "status", args: []string{"resource", "status", "example-2"}, want: "{\"name\":\"example-2\",\"status\":\"pending\"}\n"},
		{name: "apply", args: []string{"resource", "apply", "example-2"}, want: "{\"name\":\"example-2\",\"status\":\"ready\"}\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := newTestRootCommand(t)
			command.SetArgs(append([]string{"--output", "json", "--color", "always"}, test.args...))
			var output bytes.Buffer
			command.SetOut(&output)
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			if got := output.String(); got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
			if strings.Contains(output.String(), "\x1b[") {
				t.Fatalf("JSON output contains ANSI escape: %q", output.String())
			}
		})
	}
}

func TestPersistentFlagsAndValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
		bad  bool
	}{
		{name: "global flags", args: []string{"--log-level", "debug", "--output", "json", "--color", "never", "--quiet", "resource", "list"}, want: "example-1"},
		{name: "flags after resource", args: []string{"resource", "--output", "text", "list"}, want: "example-1"},
		{name: "invalid output", args: []string{"--output", "yaml", "resource", "list"}, bad: true},
		{name: "invalid log level", args: []string{"--log-level", "trace", "resource", "list"}, bad: true},
		{name: "invalid color", args: []string{"--color", "sometimes", "resource", "list"}, bad: true},
		{name: "json with forced color", args: []string{"--output", "json", "--color", "always", "resource", "list"}, want: "example-1"},
		{name: "missing name", args: []string{"resource", "status"}, bad: true},
		{name: "extra name", args: []string{"resource", "status", "example-1", "example-2"}, bad: true},
		{name: "unknown name", args: []string{"resource", "status", "missing"}, bad: true},
		{name: "extra list argument", args: []string{"resource", "list", "example-1"}, bad: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			command := newTestRootCommand(t)
			command.SetArgs(tc.args)
			var output bytes.Buffer
			command.SetOut(&output)
			err := command.Execute()
			if tc.bad && err == nil {
				t.Fatal("Execute() error = nil, want validation error")
			}
			if tc.bad && output.Len() != 0 {
				t.Fatalf("output = %q, want no output for invalid configuration or arguments", output.String())
			}
			if !tc.bad && err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if tc.want != "" && !strings.Contains(output.String(), tc.want) {
				t.Fatalf("output = %q, want it to contain %q", output.String(), tc.want)
			}
		})
	}
}

func TestResourceNameCompletion(t *testing.T) {
	command := newTestRootCommand(t)
	status, _, err := command.Find([]string{"resource", "status"})
	if err != nil {
		t.Fatal(err)
	}
	got, directive := status.ValidArgsFunction(status, nil, "example-2")
	if len(got) != 1 || got[0] != "example-2" || directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("completion = (%v, %v), want [example-2] with no file completion", got, directive)
	}
}

func TestCommandTreesHaveIndependentFlags(t *testing.T) {
	first := newTestRootCommand(t)
	second := newTestRootCommand(t)
	first.SetArgs([]string{"--output", "json", "resource", "list"})
	first.SetOut(&bytes.Buffer{})
	if err := first.Execute(); err != nil {
		t.Fatal(err)
	}
	second.SetArgs([]string{"resource", "list"})
	second.SetOut(&bytes.Buffer{})
	if err := second.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestFactoryReceivesEffectiveConfigBeforeCommand(t *testing.T) {
	setupConfigEnvironment(t)
	t.Setenv("CLI_APP_COLOR", "never")
	if err := os.Unsetenv("CLI_APP_OUTPUT"); err != nil {
		t.Fatal(err)
	}
	if err := os.Unsetenv("CLI_APP_QUIET"); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(configFile, []byte("output = 'json'\nquiet = true\n[log.console]\nlevel = 'info'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := memory.NewResourceReader()
	var got config.Config
	var gotLogger *slog.Logger
	factoryCalls := 0
	command := NewRootCommandWithFactory(func(_ context.Context, effective config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		factoryCalls++
		got = effective
		gotLogger = logger
		return app.NewResourceService(reader, logger, progressReporter), nil, nil
	}, testBuildInfo(), reader.Names())
	command.SetArgs([]string{"--config", configFile, "--log-level", "debug", "resource", "list", "--timeout", "2m"})
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	want := config.Config{
		LogLevel: "debug",
		Log:      config.LogConfig{Console: config.ConsoleLogConfig{Level: "debug"}, File: config.FileLogConfig{Level: "debug"}},
		Output:   "json",
		Color:    "never",
		Quiet:    true,
		Timeouts: config.TimeoutConfig{Cleanup: 5 * time.Second, Resource: config.ResourceTimeouts{List: durationPointer(2 * time.Minute)}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("factory config = %+v, want %+v", got, want)
	}
	if factoryCalls != 1 {
		t.Fatalf("factory calls = %d, want one call before resource execution", factoryCalls)
	}
	var results []map[string]string
	if err := json.Unmarshal(output.Bytes(), &results); err != nil || len(results) != 2 || results[0]["name"] != "example-1" {
		t.Fatalf("config-selected output = %q, want JSON resource list (decode error: %v)", output.String(), err)
	}
	if gotLogger == nil {
		t.Fatal("factory logger = nil, want initialized logger")
	}
}

func TestConfiguredResourceTimeoutReachesAdapter(t *testing.T) {
	setupConfigEnvironment(t)
	configFile := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(configFile, []byte("[timeouts.resource]\nlist = '80ms'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := blockingTimeoutReader{
		ResourceReader: memory.NewResourceReader(),
		started:        make(chan time.Time, 1),
		deadlines:      make(chan time.Time, 1),
	}
	command := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(reader, logger, progressReporter), nil, nil
	}, testBuildInfo(), memory.NewResourceReader().Names())
	parentContext, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	command.SetContext(parentContext)
	command.SetArgs([]string{"--config", configFile, "resource", "list"})
	command.SetOut(&bytes.Buffer{})
	if err := command.Execute(); !errors.Is(err, app.ErrTimeout) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Execute() error = %v, want timeout classification and context.DeadlineExceeded", err)
	}
	started := <-reader.started
	deadline := <-reader.deadlines
	if remaining := deadline.Sub(started); remaining < 50*time.Millisecond || remaining > 500*time.Millisecond {
		t.Fatalf("adapter deadline interval = %s, want configured 80ms timeout", remaining)
	}
}

func TestDependencyCleanupOnceAcrossCommandOutcomes(t *testing.T) {
	for _, test := range []struct {
		name             string
		setupFailure     bool
		operationFailure bool
	}{
		{name: "success"},
		{name: "factory setup failure", setupFailure: true},
		{name: "operation failure", operationFailure: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupConfigEnvironment(t)
			calls := 0
			var cleanupErr error
			var cleanupDeadline time.Time
			reader := app.ResourceReader(memory.NewResourceReader())
			if test.operationFailure {
				reader = failingResourceReader{}
			}
			cleanup := func(ctx context.Context) error {
				calls++
				cleanupErr = ctx.Err()
				cleanupDeadline, _ = ctx.Deadline()
				return nil
			}
			command := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
				if test.setupFailure {
					return nil, cleanup, errors.New("injected setup failure")
				}
				return app.NewResourceService(reader, logger, progressReporter), cleanup, nil
			}, testBuildInfo(), memory.NewResourceReader().Names())
			command.SetArgs([]string{"resource", "list"})
			command.SetOut(&bytes.Buffer{})
			err := command.Execute()
			if test.setupFailure || test.operationFailure {
				if err == nil {
					t.Fatal("Execute() error = nil, want failure")
				}
			} else if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if calls != 1 {
				t.Fatalf("cleanup calls = %d, want one", calls)
			}
			if cleanupErr != nil || cleanupDeadline.IsZero() || time.Until(cleanupDeadline) <= 0 {
				t.Fatalf("cleanup context err/deadline = %v/%v, want live bounded context", cleanupErr, cleanupDeadline)
			}
		})
	}
}

func TestDependencyCleanupGetsFreshBoundedContextAfterOperationCancellation(t *testing.T) {
	setupConfigEnvironment(t)
	configFile := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(configFile, []byte("[timeouts]\ncleanup = '300ms'\n[timeouts.resource]\nlist = '30ms'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := blockingTimeoutReader{
		ResourceReader: memory.NewResourceReader(),
		started:        make(chan time.Time, 1),
		deadlines:      make(chan time.Time, 1),
	}
	type cleanupContextState struct {
		err      error
		deadline time.Time
		bounded  bool
	}
	cleanupContext := make(chan cleanupContextState, 1)
	cleanupCalls := 0
	command := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(reader, logger, progressReporter), func(ctx context.Context) error {
			cleanupCalls++
			deadline, bounded := ctx.Deadline()
			cleanupContext <- cleanupContextState{err: ctx.Err(), deadline: deadline, bounded: bounded}
			return nil
		}, nil
	}, testBuildInfo(), memory.NewResourceReader().Names())
	command.SetArgs([]string{"--config", configFile, "resource", "list"})
	command.SetOut(&bytes.Buffer{})
	if err := command.Execute(); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Execute() error = %v, want operation deadline", err)
	}
	if cleanupCalls != 1 {
		t.Fatalf("cleanup calls = %d, want one", cleanupCalls)
	}
	state := <-cleanupContext
	if state.err != nil {
		t.Fatalf("cleanup context error = %v, want live context", state.err)
	}
	if !state.bounded {
		t.Fatal("cleanup context has no deadline")
	}
	if remaining := time.Until(state.deadline); remaining < 100*time.Millisecond || remaining > 300*time.Millisecond {
		t.Fatalf("cleanup deadline remaining = %s, want configured 300ms bound", remaining)
	}
}

type blockingTimeoutReader struct {
	app.ResourceReader
	started   chan time.Time
	deadlines chan time.Time
}

func (r blockingTimeoutReader) List(ctx context.Context) ([]domain.Resource, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil, errors.New("operation context has no deadline")
	}
	r.started <- time.Now()
	r.deadlines <- deadline
	<-ctx.Done()
	return nil, ctx.Err()
}

func durationPointer(duration time.Duration) *time.Duration { return &duration }

func TestLoggingInitializationFailurePreventsFactory(t *testing.T) {
	setupConfigEnvironment(t)
	configFile := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(configFile, []byte("[log.file]\npath = 'missing/app.jsonl'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	factoryCalls := 0
	reader := memory.NewResourceReader()
	command := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		factoryCalls++
		return app.NewResourceService(reader, logger, progressReporter), nil, nil
	}, testBuildInfo(), reader.Names())
	command.SetArgs([]string{"--config", configFile, "resource", "list"})
	if err := command.Execute(); err == nil || !strings.Contains(err.Error(), "open log file") {
		t.Fatalf("Execute() error = %v, want log file initialization error", err)
	}
	if factoryCalls != 0 {
		t.Fatalf("factory calls = %d, want zero on logging initialization failure", factoryCalls)
	}
}

func TestLoggerStaysOffCommandOutput(t *testing.T) {
	setupConfigEnvironment(t)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previousStderr := os.Stderr
	os.Stderr = writer
	commandReader := memory.NewResourceReader()
	command := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(commandReader, logger, progressReporter), nil, nil
	}, testBuildInfo(), commandReader.Names())
	command.SetArgs([]string{"--log-level", "debug", "--output", "json", "resource", "list"})
	var stdout bytes.Buffer
	var progress bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&progress)
	listCommand, _, err := command.Find([]string{"resource", "list"})
	if err != nil {
		t.Fatal(err)
	}
	listCommand.PreRunE = func(cmd *cobra.Command, _ []string) error {
		_, err := fmt.Fprintln(cmd.ErrOrStderr(), "progress update")
		return err
	}
	executeErr := command.Execute()
	os.Stderr = previousStderr
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	stderr, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if executeErr != nil {
		t.Fatal(executeErr)
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	var results []map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil || len(results) != 2 || results[0]["name"] != "example-1" {
		t.Fatalf("command output = %q, want parseable JSON results (decode error: %v)", stdout.String(), err)
	}
	if !strings.Contains(string(stderr), "listing resources") || strings.Contains(string(stderr), "example-1") {
		t.Fatalf("diagnostics = %q, want log record only", stderr)
	}
	if progress.String() != "progress update\n" || strings.Contains(stdout.String(), "progress update") {
		t.Fatalf("progress = %q, stdout = %q; want progress on stderr only", progress.String(), stdout.String())
	}
}

func TestApplyProgressRespectsQuietAndKeepsJSONResult(t *testing.T) {
	for _, test := range []struct {
		name  string
		quiet bool
		want  string
	}{
		{name: "progress to redirected stderr", want: "validating: Validating resource\napplying: Applying resource\ncomplete: Resource applied (1/1)\n"},
		{name: "quiet suppresses progress", quiet: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupConfigEnvironment(t)
			command := NewRootCommand(memory.NewResourceReader(), testBuildInfo(), memory.NewResourceReader().Names())
			args := []string{"--output", "json"}
			if test.quiet {
				args = append(args, "--quiet")
			}
			args = append(args, "resource", "apply", "example-1")
			command.SetArgs(args)
			var stdout, stderr bytes.Buffer
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			var result map[string]string
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatalf("stdout = %q, want JSON result: %v", stdout.String(), err)
			}
			if len(result) != 2 || result["name"] != "example-1" || result["status"] != "ready" {
				t.Fatalf("JSON result = %#v, want unwrapped resource result", result)
			}
			if got := stderr.String(); got != test.want {
				t.Fatalf("stderr = %q, want %q", got, test.want)
			}
		})
	}
}

func TestQuietDoesNotSuppressApplyErrors(t *testing.T) {
	setupConfigEnvironment(t)
	command := NewRootCommand(failingResourceReader{}, testBuildInfo(), memory.NewResourceReader().Names())
	command.SetArgs([]string{"--quiet", "resource", "apply", "example-1"})
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	if err := command.Execute(); err == nil || !strings.Contains(err.Error(), "injected reader failure") {
		t.Fatalf("Execute() error = %v, want injected reader failure", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout/stderr = %q/%q, want no partial result or progress", stdout.String(), stderr.String())
	}
}

func TestRendererWriteErrorsAreReturned(t *testing.T) {
	wantErr := errors.New("injected write failure")
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			command := newTestRootCommand(t)
			command.SetArgs([]string{"--output", format, "resource", "list"})
			command.SetOut(failingWriter{err: wantErr})
			if err := command.Execute(); !errors.Is(err, wantErr) {
				t.Fatalf("Execute() error = %v, want writer error", err)
			}
		})
	}
}

func TestConfiguredFilePathAndCleanupAcrossCommandResults(t *testing.T) {
	for _, failCommand := range []bool{false, true} {
		name := "success"
		if failCommand {
			name = "command error"
		}
		t.Run(name, func(t *testing.T) {
			setupConfigEnvironment(t)
			root := t.TempDir()
			if err := os.Mkdir(filepath.Join(root, "logs"), 0o700); err != nil {
				t.Fatal(err)
			}
			fixture, err := os.ReadFile("testdata/log-file.toml")
			if err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(root, "settings.toml")
			if err := os.WriteFile(configPath, fixture, 0o600); err != nil {
				t.Fatal(err)
			}
			reader := app.ResourceReader(memory.NewResourceReader())
			if failCommand {
				reader = failingResourceReader{}
			}
			command := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
				return app.NewResourceService(reader, logger, progressReporter), nil, nil
			}, testBuildInfo(), memory.NewResourceReader().Names())
			command.SetArgs([]string{"--config", configPath, "resource", "list"})
			command.SetOut(&bytes.Buffer{})
			err = command.Execute()
			if failCommand && (err == nil || !strings.Contains(err.Error(), "injected reader failure")) {
				t.Fatalf("Execute() error = %v, want injected reader error", err)
			}
			if !failCommand && err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			logPath := filepath.Join(root, "logs", "cli-app.jsonl")
			file, err := os.Open(logPath)
			if err != nil {
				t.Fatalf("open source-relative log file: %v", err)
			}
			scanner := bufio.NewScanner(file)
			var records int
			for scanner.Scan() {
				var record map[string]any
				if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
					t.Fatalf("JSONL record %q: %v", scanner.Text(), err)
				}
				if record["msg"] != "listing resources" {
					t.Errorf("log record = %#v, want application debug record", record)
				}
				records++
			}
			if err := scanner.Err(); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			if records != 1 {
				t.Fatalf("JSONL records = %d, want 1", records)
			}
			if err := os.Remove(logPath); err != nil {
				t.Fatalf("remove closed log file: %v", err)
			}
		})
	}
}

type failingResourceReader struct{}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func (failingResourceReader) List(context.Context) ([]domain.Resource, error) {
	return nil, errors.New("injected reader failure")
}

func (failingResourceReader) Status(context.Context, string) (domain.Resource, error) {
	return domain.Resource{}, errors.New("injected reader failure")
}

func (failingResourceReader) Apply(context.Context, string) (domain.Resource, error) {
	return domain.Resource{}, errors.New("injected reader failure")
}

func TestFactoryErrorPropagates(t *testing.T) {
	setupConfigEnvironment(t)
	wantErr := errors.New("factory failure")
	command := NewRootCommandWithFactory(func(context.Context, config.Config, *slog.Logger, app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		return nil, nil, wantErr
	}, testBuildInfo(), nil)
	command.SetArgs([]string{"resource", "list"})

	if err := command.Execute(); !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want wrapped factory error", err)
	}
}

func TestHelpAndCompletionSkipConfigurationAndFactory(t *testing.T) {
	setupConfigEnvironment(t)
	for _, args := range [][]string{
		{"--config", filepath.Join(t.TempDir(), "missing.toml"), "--help"},
		{"--config", filepath.Join(t.TempDir(), "missing.toml"), "completion", "bash"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			factoryCalls := 0
			command := NewRootCommandWithFactory(func(context.Context, config.Config, *slog.Logger, app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
				factoryCalls++
				return nil, nil, errors.New("factory must not run")
			}, testBuildInfo(), nil)
			command.SetArgs(args)
			command.SetOut(&bytes.Buffer{})
			if err := command.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if factoryCalls != 0 {
				t.Fatalf("factory calls = %d, want zero", factoryCalls)
			}
		})
	}
}

func TestRepeatedExecutionReloadsConfigAndDependencies(t *testing.T) {
	setupConfigEnvironment(t)
	if err := os.Unsetenv("CLI_APP_LOG_LEVEL"); err != nil {
		t.Fatal(err)
	}
	firstConfig := filepath.Join(t.TempDir(), "first.toml")
	secondConfig := filepath.Join(t.TempDir(), "second.toml")
	if err := os.WriteFile(firstConfig, []byte("[log.console]\nlevel = 'info'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondConfig, []byte("[log.console]\nlevel = 'error'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := memory.NewResourceReader()
	var configs []config.Config
	command := NewRootCommandWithFactory(func(_ context.Context, effective config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		configs = append(configs, effective)
		return app.NewResourceService(reader, logger, progressReporter), nil, nil
	}, testBuildInfo(), reader.Names())
	for _, path := range []string{firstConfig, secondConfig} {
		command.SetArgs([]string{"--config", path, "resource", "list"})
		command.SetOut(&bytes.Buffer{})
		if err := command.Execute(); err != nil {
			t.Fatalf("Execute() with %s: %v", path, err)
		}
	}
	if len(configs) != 2 || configs[0].LogLevel != "info" || configs[1].LogLevel != "error" {
		t.Fatalf("factory configs = %+v, want info then error", configs)
	}
}

func setupConfigEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "no-user-config"))
	t.Setenv("CLI_APP_LOG_LEVEL", "warn")
	t.Setenv("CLI_APP_OUTPUT", "text")
	t.Setenv("CLI_APP_COLOR", "auto")
	t.Setenv("CLI_APP_QUIET", "false")
}

func testBuildInfo() BuildInfo {
	return BuildInfo{Version: "1.2.3", Commit: "abc123", BuildDate: "test-build"}
}

func newTestRootCommand(t *testing.T) *cobra.Command {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "no-user-config"))
	for name, value := range map[string]string{
		"CLI_APP_LOG_LEVEL": "warn",
		"CLI_APP_OUTPUT":    "text",
		"CLI_APP_COLOR":     "auto",
		"CLI_APP_QUIET":     "false",
	} {
		t.Setenv(name, value)
	}
	reader := memory.NewResourceReader()
	return NewRootCommand(reader, testBuildInfo(), reader.Names())
}
