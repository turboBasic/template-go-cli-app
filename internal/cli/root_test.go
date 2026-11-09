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
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
	"github.com/turboBasic/template-go-cli-app/internal/domain"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
)

func TestResourceListCommand(t *testing.T) {
	command := newTestRootCommand(t)
	command.SetArgs([]string{"demo", "structured-output"})
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got [][]string
	for line := range strings.Lines(output.String()) {
		got = append(got, strings.Fields(line))
	}
	want := [][]string{{"NAME", "STATUS"}, {"example-1", "ready"}, {"example-2", "pending"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rows = %q, want %q", got, want)
	}
}

func TestRootFeatures(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"help", "demo", "not-found"}, {"--version"}, {"completion", "bash"}, {"completion", "zsh"}, {"completion", "fish"}, {"completion", "powershell"}} {
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

// resultWords is a human result's words, so its punctuation and spacing may change without breaking a test.
func resultWords(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' })
}

func TestResourceStatusAndApplyCommands(t *testing.T) {
	for _, tc := range []struct {
		args string
		want []string
	}{
		{args: "not-found example-2", want: []string{"example-2", "pending"}},
		{args: "secret-credential example-2", want: []string{"example-2", "ready"}},
	} {
		t.Run(tc.args, func(t *testing.T) {
			command := newTestRootCommand(t)
			command.SetArgs(strings.Fields("demo " + tc.args))
			command.SetIn(bytes.NewBuffer(nil))
			var output bytes.Buffer
			command.SetOut(&output)
			if err := command.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if got := resultWords(output.String()); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("result words = %q, want %q", got, tc.want)
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
		{name: "list", args: []string{"demo", "structured-output"}, want: "[{\"name\":\"example-1\",\"status\":\"ready\"},{\"name\":\"example-2\",\"status\":\"pending\"}]\n"},
		{name: "status", args: []string{"demo", "not-found", "example-2"}, want: "{\"name\":\"example-2\",\"status\":\"pending\"}\n"},
		{name: "apply", args: []string{"demo", "secret-credential", "example-2"}, want: "{\"name\":\"example-2\",\"status\":\"ready\"}\n"},
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
			// Decoding also fails on a stray ANSI escape, which JSON must never carry.
			var got, want any
			if err := json.Unmarshal(output.Bytes(), &got); err != nil {
				t.Fatalf("output %q is not JSON: %v", output.String(), err)
			}
			if err := json.Unmarshal([]byte(test.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("output = %q, want JSON equal to %q", output.String(), test.want)
			}
		})
	}
}

func TestResourceResultColor(t *testing.T) {
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	run := func(t *testing.T, flags []string, command []string) string {
		t.Helper()
		root := newTestRootCommand(t)
		root.SetArgs(append(append([]string{}, flags...), command...))
		var output bytes.Buffer
		root.SetOut(&output)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		return output.String()
	}
	for _, command := range [][]string{
		{"demo", "structured-output"},
		{"demo", "not-found", "example-2"},
		{"demo", "secret-credential", "example-2"},
	} {
		t.Run(strings.Join(command[1:], " "), func(t *testing.T) {
			redirected := run(t, nil, command)
			plainJSON := run(t, []string{"--output", "json"}, command)
			tests := []struct {
				name   string
				flags  []string
				styled bool
				want   string
			}{
				{name: "redirected default", want: redirected},
				{name: "json always", flags: []string{"--output", "json", "--color", "always"}, want: plainJSON},
				{name: "never", flags: []string{"--color", "never"}, want: redirected},
				{name: "always", flags: []string{"--color", "always"}, styled: true},
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					got := run(t, test.flags, command)
					if has := strings.Contains(got, "\x1b"); has != test.styled {
						t.Fatalf("output = %q, ESC present = %t, want %t", got, has, test.styled)
					}
					if test.styled {
						if stripped := ansi.ReplaceAllString(got, ""); stripped != redirected {
							t.Fatalf("stripped = %q, want redirected %q", stripped, redirected)
						}
					}
					if test.want != "" && got != test.want {
						t.Fatalf("output = %q, want %q", got, test.want)
					}
				})
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
		{name: "global flags", args: []string{"--log-level", "debug", "--output", "json", "--color", "never", "--quiet", "demo", "structured-output"}, want: "example-1"},
		{name: "flags after demo", args: []string{"demo", "--output", "text", "structured-output"}, want: "example-1"},
		{name: "invalid output", args: []string{"--output", "yaml", "demo", "structured-output"}, bad: true},
		{name: "invalid log level", args: []string{"--log-level", "trace", "demo", "structured-output"}, bad: true},
		{name: "invalid color", args: []string{"--color", "sometimes", "demo", "structured-output"}, bad: true},
		{name: "json with forced color", args: []string{"--output", "json", "--color", "always", "demo", "structured-output"}, want: "example-1"},
		{name: "missing name", args: []string{"demo", "not-found"}, bad: true},
		{name: "extra name", args: []string{"demo", "not-found", "example-1", "example-2"}, bad: true},
		{name: "unknown name", args: []string{"demo", "not-found", "missing"}, bad: true},
		{name: "extra list argument", args: []string{"demo", "structured-output", "example-1"}, bad: true},
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

func TestPersistentColorFlagContract(t *testing.T) {
	flag := newTestRootCommand(t).PersistentFlags().Lookup("color")
	if flag == nil {
		t.Fatal("--color flag is not registered")
	}
	if flag.DefValue != "auto" {
		t.Errorf("--color default = %q, want auto", flag.DefValue)
	}
	for _, want := range []string{"progress", "auto", "always", "never"} {
		if !strings.Contains(flag.Usage, want) {
			t.Errorf("--color usage = %q, want it to mention %q", flag.Usage, want)
		}
	}
}

func TestResourceNameCompletion(t *testing.T) {
	command := newTestRootCommand(t)
	status, _, err := command.Find([]string{"demo", "not-found"})
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
	first.SetArgs([]string{"--output", "json", "demo", "structured-output"})
	first.SetOut(&bytes.Buffer{})
	if err := first.Execute(); err != nil {
		t.Fatal(err)
	}
	second.SetArgs([]string{"demo", "structured-output"})
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
	if err := os.WriteFile(configFile, []byte("output = 'json'\nquiet = true\n[log]\nlevel = 'info'\n"), 0o600); err != nil {
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
	command.SetArgs([]string{"--config", configFile, "--log-level", "debug", "demo", "structured-output", "--timeout", "2m"})
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	want := config.Config{
		Log:      config.LogConfig{Level: "debug", File: config.FileLogConfig{Level: "debug"}},
		Output:   "json",
		Color:    "never",
		Quiet:    true,
		Timeouts: config.TimeoutConfig{Cleanup: 5 * time.Second},
		Retry:    config.Defaults().Retry,
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
	if err := os.WriteFile(configFile, []byte("[timeouts.demo]\nstructured_output = '80ms'\n"), 0o600); err != nil {
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
	command.SetArgs([]string{"--config", configFile, "demo", "structured-output"})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(io.Discard)
	if err := Execute(context.Background(), command); !errors.Is(err, app.ErrTimeout) || !errors.Is(err, context.DeadlineExceeded) {
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
			command.SetArgs([]string{"demo", "structured-output"})
			command.SetOut(&bytes.Buffer{})
			command.SetErr(io.Discard)
			err := Execute(context.Background(), command)
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

func TestExecuteCleansUpAfterNonDemoCommandFailure(t *testing.T) {
	setupConfigEnvironment(t)
	commandErr := errors.New("ordinary command failed")
	cleanupErr := errors.New("dependency cleanup failed")
	cleanupCalls := 0
	reader := memory.NewResourceReader()
	root := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(reader, logger, progressReporter), func(context.Context) error {
			cleanupCalls++
			return cleanupErr
		}, nil
	}, testBuildInfo(), reader.Names())
	root.AddCommand(&cobra.Command{Use: "ordinary", RunE: func(*cobra.Command, []string) error { return commandErr }})
	root.SetArgs([]string{"ordinary"})
	root.SetErr(io.Discard)

	err := Execute(context.Background(), root)
	if !errors.Is(err, commandErr) || !errors.Is(err, cleanupErr) {
		t.Fatalf("Execute() error = %v, want command and cleanup errors", err)
	}
	if cleanupCalls != 1 {
		t.Fatalf("cleanup calls = %d, want one", cleanupCalls)
	}
}

func TestExecuteCleansUpAfterCommandUsingRun(t *testing.T) {
	setupConfigEnvironment(t)
	cleanupCalls := 0
	reader := memory.NewResourceReader()
	root := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(reader, logger, progressReporter), func(context.Context) error {
			cleanupCalls++
			return nil
		}, nil
	}, testBuildInfo(), reader.Names())
	root.AddCommand(&cobra.Command{Use: "ordinary", Run: func(*cobra.Command, []string) {}})
	root.SetArgs([]string{"ordinary"})
	root.SetErr(io.Discard)

	if err := Execute(context.Background(), root); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if cleanupCalls != 1 {
		t.Fatalf("cleanup calls = %d, want one", cleanupCalls)
	}
}

func TestGlobalTimeoutAppliesWithoutConfiguredOperationTimeout(t *testing.T) {
	setupConfigEnvironment(t)
	reader := memory.NewResourceReader()
	root := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(reader, logger, progressReporter), nil, nil
	}, testBuildInfo(), reader.Names())
	root.AddCommand(&cobra.Command{Use: "ordinary", RunE: func(cmd *cobra.Command, _ []string) error {
		if _, ok := cmd.Context().Deadline(); !ok {
			return errors.New("operation context has no deadline")
		}
		<-cmd.Context().Done()
		return cmd.Context().Err()
	}})
	root.SetArgs([]string{"--timeout", "20ms", "ordinary"})
	root.SetErr(io.Discard)

	if err := Execute(context.Background(), root); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Execute() error = %v, want command deadline", err)
	}
}

func TestExplicitTimeoutOverridesConfiguredOperationTimeout(t *testing.T) {
	setupConfigEnvironment(t)
	configFile := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(configFile, []byte("[timeouts.demo]\nstructured_output = '2s'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := blockingTimeoutReader{
		ResourceReader: memory.NewResourceReader(),
		started:        make(chan time.Time, 1),
		deadlines:      make(chan time.Time, 1),
	}
	root := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(reader, logger, progressReporter), nil, nil
	}, testBuildInfo(), memory.NewResourceReader().Names())
	root.SetArgs([]string{"--config", configFile, "--timeout", "30ms", "demo", "structured-output"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	if err := Execute(context.Background(), root); !errors.Is(err, app.ErrTimeout) {
		t.Fatalf("Execute() error = %v, want operation timeout", err)
	}
	started := <-reader.started
	deadline := <-reader.deadlines
	if remaining := deadline.Sub(started); remaining < 10*time.Millisecond || remaining > 300*time.Millisecond {
		t.Fatalf("adapter deadline interval = %s, want explicit 30ms timeout instead of configured 2s", remaining)
	}
}

func TestDependencyCleanupGetsFreshBoundedContextAfterOperationCancellation(t *testing.T) {
	setupConfigEnvironment(t)
	configFile := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(configFile, []byte("[timeouts]\ncleanup = '300ms'\n[timeouts.demo]\nstructured_output = '30ms'\n"), 0o600); err != nil {
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
	command.SetArgs([]string{"--config", configFile, "demo", "structured-output"})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(io.Discard)
	if err := Execute(context.Background(), command); !errors.Is(err, context.DeadlineExceeded) {
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
	command.SetArgs([]string{"--config", configFile, "demo", "structured-output"})
	if err := command.Execute(); err == nil || !strings.Contains(err.Error(), "open log file") {
		t.Fatalf("Execute() error = %v, want log file initialization error", err)
	}
	if factoryCalls != 0 {
		t.Fatalf("factory calls = %d, want zero on logging initialization failure", factoryCalls)
	}
}

func TestLoggerStaysOffCommandOutput(t *testing.T) {
	setupConfigEnvironment(t)
	reader := memory.NewResourceReader()
	command := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(reader, logger, progressReporter), nil, nil
	}, testBuildInfo(), reader.Names())
	command.SetArgs([]string{"--log-level", "debug", "--output", "json", "demo", "structured-output"})
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	var results []map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil || len(results) != 2 || results[0]["name"] != "example-1" {
		t.Fatalf("command output = %q, want parseable JSON results (decode error: %v)", stdout.String(), err)
	}
	if !strings.Contains(stderr.String(), "listing resources") || strings.Contains(stderr.String(), "example-1") {
		t.Fatalf("diagnostics = %q, want log record only", stderr.String())
	}
}

func TestApplyProgressRespectsQuietAndKeepsJSONResult(t *testing.T) {
	for _, test := range []struct {
		name  string
		quiet bool
	}{
		{name: "progress to redirected stderr"},
		{name: "quiet suppresses progress", quiet: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupConfigEnvironment(t)
			command := newTestRootCommand(t)
			args := []string{"--output", "json"}
			if test.quiet {
				args = append(args, "--quiet")
			}
			args = append(args, "demo", "secret-credential", "example-1")
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
			got := stderr.String()
			if test.quiet && got != "" || !test.quiet && (!strings.Contains(got, "complete: ") || strings.Contains(got, "\x1b")) {
				t.Fatalf("stderr = %q, quiet = %t; want empty when quiet, plain progress otherwise", got, test.quiet)
			}
		})
	}
}

func TestApplyProgressColor(t *testing.T) {
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	for _, test := range []struct {
		name string
		args []string
		mode string // plain, quiet or styled
	}{
		{name: "redirected default", mode: "plain"},
		{name: "never", args: []string{"--color", "never"}, mode: "plain"},
		{name: "json with always styles stderr progress", args: []string{"--output", "json", "--color", "always"}, mode: "styled"},
		{name: "quiet with always", args: []string{"--quiet", "--color", "always"}, mode: "quiet"},
		{name: "always", args: []string{"--color", "always"}, mode: "styled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupConfigEnvironment(t)
			command := newTestRootCommand(t)
			command.SetArgs(append(append([]string{}, test.args...), "demo", "secret-credential", "example-2"))
			var stdout, stderr bytes.Buffer
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			got := stderr.String()
			switch test.mode {
			case "quiet":
				if got != "" {
					t.Fatalf("stderr = %q, want none", got)
				}
			case "plain":
				if !strings.Contains(got, "complete: ") || strings.Contains(got, "\x1b") {
					t.Fatalf("stderr = %q, want plain progress without escapes", got)
				}
			default:
				if !strings.Contains(got, "\x1b[") || !strings.Contains(ansi.ReplaceAllString(got, ""), "✓ ") {
					t.Fatalf("stderr = %q, want ANSI escapes and a completion mark", got)
				}
			}
		})
	}
}

func TestQuietDoesNotSuppressApplyErrors(t *testing.T) {
	setupConfigEnvironment(t)
	command := newTestRootCommandWithReader(t, failingResourceReader{})
	command.SetArgs([]string{"--quiet", "demo", "secret-credential", "example-1"})
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

func TestProgressWriterErrorsDoNotReplaceApplyResult(t *testing.T) {
	setupConfigEnvironment(t)
	reader := memory.NewResourceReader()
	command := newTestRootCommandWithReader(t, reader)
	command.SetArgs([]string{"--output", "json", "demo", "secret-credential", "example-1"})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(failingWriter{err: errors.New("injected progress write failure")})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, want successful operation despite progress write failure", err)
	}
	var result map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout = %q, want JSON result: %v", stdout.String(), err)
	}
	if len(result) != 2 || result["name"] != "example-1" || result["status"] != "ready" {
		t.Fatalf("JSON result = %#v, want unwrapped applied resource", result)
	}
}

func TestRendererWriteErrorsAreReturned(t *testing.T) {
	wantErr := errors.New("injected write failure")
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			command := newTestRootCommand(t)
			command.SetArgs([]string{"--output", format, "demo", "structured-output"})
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
			command.SetArgs([]string{"--config", configPath, "demo", "structured-output"})
			command.SetOut(&bytes.Buffer{})
			command.SetErr(io.Discard)
			err = Execute(context.Background(), command)
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

func (failingResourceReader) Apply(context.Context, string, app.Secret) (domain.Resource, error) {
	return domain.Resource{}, errors.New("injected reader failure")
}

func TestFactoryErrorPropagates(t *testing.T) {
	setupConfigEnvironment(t)
	wantErr := errors.New("factory failure")
	command := NewRootCommandWithFactory(func(context.Context, config.Config, *slog.Logger, app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		return nil, nil, wantErr
	}, testBuildInfo(), nil)
	command.SetArgs([]string{"demo", "structured-output"})

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

func TestConfigShowSkipsLoggingAndFactory(t *testing.T) {
	setupConfigEnvironment(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "cli.toml")
	if err := os.WriteFile(path, []byte("[log.file]\npath = 'app.jsonl'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		args []string
		want error
	}{
		{"succeeds", []string{"--config", path, "config", "show"}, nil},
		{"invalid log level", []string{"--config", path, "--log-level", "nope", "config", "show"}, app.ErrValidation},
	} {
		t.Run(test.name, func(t *testing.T) {
			factoryCalls := 0
			command := NewRootCommandWithFactory(func(context.Context, config.Config, *slog.Logger, app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
				factoryCalls++
				return nil, nil, errors.New("factory must not run")
			}, testBuildInfo(), nil)
			command.SetArgs(test.args)
			command.SetOut(&bytes.Buffer{})
			if err := command.Execute(); !errors.Is(err, test.want) {
				t.Fatalf("Execute() error = %v, want %v", err, test.want)
			}
			if factoryCalls != 0 {
				t.Fatalf("factory calls = %d, want zero", factoryCalls)
			}
			if _, err := os.Stat(filepath.Join(dir, "app.jsonl")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("log file stat error = %v, want not exist", err)
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
	if err := os.WriteFile(firstConfig, []byte("[log]\nlevel = 'info'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondConfig, []byte("[log]\nlevel = 'error'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := memory.NewResourceReader()
	var configs []config.Config
	command := NewRootCommandWithFactory(func(_ context.Context, effective config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		configs = append(configs, effective)
		return app.NewResourceService(reader, logger, progressReporter), nil, nil
	}, testBuildInfo(), reader.Names())
	for _, path := range []string{firstConfig, secondConfig} {
		command.SetArgs([]string{"--config", path, "demo", "structured-output"})
		command.SetOut(&bytes.Buffer{})
		if err := command.Execute(); err != nil {
			t.Fatalf("Execute() with %s: %v", path, err)
		}
	}
	if len(configs) != 2 || configs[0].Log.Level != "info" || configs[1].Log.Level != "error" {
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
	return newTestRootCommandWithReader(t, memory.NewResourceReader())
}

func newTestRootCommandWithReader(t *testing.T, reader app.ResourceReader) *cobra.Command {
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
	return NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(reader, logger, progressReporter), nil, nil
	}, testBuildInfo(), memory.NewResourceReader().Names())
}

type countingResolver struct{ calls int }

func (r *countingResolver) Resolve(context.Context, string) (app.Secret, error) {
	r.calls++
	return "resolved-secret", nil
}

func TestCredentialResolvesOnlyWhenApplying(t *testing.T) {
	for _, test := range []struct {
		name      string
		args      []string
		wantCalls int
	}{
		{name: "help", args: []string{"--help"}},
		{name: "completion", args: []string{"completion", "bash"}},
		{name: "config show", args: []string{"config", "show"}},
		{name: "demo structured-output", args: []string{"demo", "structured-output"}},
		{name: "demo not-found", args: []string{"demo", "not-found", "example-1"}},
		{name: "demo secret-credential", args: []string{"demo", "secret-credential", "example-1"}, wantCalls: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupConfigEnvironment(t)
			t.Setenv("CLI_APP_DEMO_SECRET_CREDENTIAL_REFERENCE", "op://vault/item/field")
			resolver := &countingResolver{}
			reader := memory.NewResourceReader()
			command := NewRootCommandWithFactory(func(_ context.Context, effective config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
				return app.NewResourceService(reader, logger, progressReporter, app.WithCredential(resolver, effective.Demo.SecretCredential.Reference)), nil, nil
			}, testBuildInfo(), reader.Names())
			var out bytes.Buffer
			command.SetOut(&out)
			command.SetErr(&bytes.Buffer{})
			command.SetArgs(append([]string{"--quiet"}, test.args...))

			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			if resolver.calls != test.wantCalls {
				t.Fatalf("resolver calls = %d, want %d", resolver.calls, test.wantCalls)
			}
			if test.name == "config show" && (!strings.Contains(out.String(), "op://vault/item/field") || strings.Contains(out.String(), "resolved-secret")) {
				t.Fatalf("config show output = %q, want the reference and not the resolved value", out.String())
			}
		})
	}
}

func TestDemoLongRunningProgress(t *testing.T) {
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	for _, test := range []struct {
		name string
		args []string
		mode string // plain, quiet or styled
	}{
		{name: "redirected default", mode: "plain"},
		{name: "never", args: []string{"--color", "never"}, mode: "plain"},
		{name: "json with always styles stderr progress", args: []string{"--output", "json", "--color", "always"}, mode: "styled"},
		{name: "quiet with always", args: []string{"--quiet", "--color", "always"}, mode: "quiet"},
		{name: "always", args: []string{"--color", "always"}, mode: "styled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupConfigEnvironment(t)
			reader := memory.NewResourceReader()
			command := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
				return app.NewResourceService(reader, logger, progressReporter, app.WithLongRunningDuration(20*time.Millisecond)), nil, nil
			}, testBuildInfo(), reader.Names())
			command.SetArgs(append(append([]string{}, test.args...), "demo", "long-running"))
			var stdout, stderr bytes.Buffer
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want none", stdout.String())
			}
			got := stderr.String()
			switch test.mode {
			case "quiet":
				if got != "" {
					t.Fatalf("stderr = %q, want none", got)
				}
			case "plain":
				if !strings.Contains(got, "Working") || !regexp.MustCompile(`complete: .*\(\d+/\d+\)\n$`).MatchString(got) || strings.Contains(got, "\x1b") {
					t.Fatalf("stderr = %q, want plain progress and counted completion, with no escapes", got)
				}
			default:
				plain := ansi.ReplaceAllString(got, "")
				if !strings.Contains(got, "\x1b[") || !regexp.MustCompile(`\d+%.*Working`).MatchString(plain) || !regexp.MustCompile(`\(\d+/\d+\)\n$`).MatchString(plain) {
					t.Fatalf("stderr = %q, want styled progress and counted completion", got)
				}
			}
		})
	}
}

func TestDemoCommands(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
	}{
		{name: "group help", args: []string{"demo"}},
		{name: "help topic", args: []string{"help", "demo", "long-running"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := newTestRootCommand(t)
			command.SetArgs(test.args)
			var stdout, stderr bytes.Buffer
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			err := command.Execute()
			if err != nil || !strings.Contains(stdout.String(), "long-running") {
				t.Fatalf("Execute() error = %v, stdout %q; want help naming long-running", err, stdout.String())
			}
		})
	}
}

func TestDemoLongRunningHonorsTimeout(t *testing.T) {
	setupConfigEnvironment(t)
	reader := memory.NewResourceReader()
	command := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(reader, logger, progressReporter, app.WithLongRunningDuration(time.Minute)), nil, nil
	}, testBuildInfo(), reader.Names())
	command.SetArgs([]string{"demo", "long-running", "--timeout", "50ms"})
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	started := time.Now()
	if err := command.Execute(); !errors.Is(err, app.ErrTimeout) {
		t.Fatalf("Execute() error = %v, want a timeout error", err)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second || strings.Contains(stderr.String(), "complete:") {
		t.Fatalf("took %s with stderr %q, want it to stop at the timeout without completing", elapsed, stderr.String())
	}
}

func TestDemoCommandsAreDemonstrations(t *testing.T) {
	demo, _, err := newTestRootCommand(t).Find([]string{"demo"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, command := range demo.Commands() {
		got = append(got, command.Name())
		if !strings.HasPrefix(command.Short, "Demonstrate") {
			t.Errorf("demo %s Short = %q, want it to begin Demonstrate", command.Name(), command.Short)
		}
		if command.Long == "" {
			t.Errorf("demo %s has no Long saying what to look at", command.Name())
		}
	}
	slices.Sort(got)
	if want := []string{"flaky-dependency", "long-running", "not-found", "secret-credential", "structured-output"}; !slices.Equal(got, want) {
		t.Errorf("demo subcommands = %v, want %v", got, want)
	}
}

func TestFlakyDependencyHelpStatesTheDefaults(t *testing.T) {
	reader := memory.NewResourceReader()
	command := newTestRootCommandWithReader(t, reader)
	demo, _, err := command.Find([]string{"demo", "flaky-dependency"})
	if err != nil {
		t.Fatal(err)
	}
	policy := config.Defaults().Retry.Demo.FlakyDependency
	for _, want := range []string{
		fmt.Sprintf("fails %d times", app.FlakyFailures),
		fmt.Sprintf("allows %d attempts, waits %s", policy.MaxAttempts, policy.InitialDelay),
		fmt.Sprintf("up to %s", policy.MaxDelay),
	} {
		if !strings.Contains(demo.Long, want) {
			t.Errorf("Long = %q, want it to contain %q", demo.Long, want)
		}
	}
}

func TestFlakyDependencyCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr error
	}{
		{name: "the first failure is final without a retry policy", args: []string{"demo", "flaky-dependency"}, wantErr: app.ErrUnavailable},
		{name: "a non-positive timeout is a validation error", args: []string{"demo", "flaky-dependency", "--timeout", "0s"}, wantErr: app.ErrValidation},
		{name: "the timeout bounds the demo", args: []string{"demo", "flaky-dependency", "--timeout", "1ns"}, wantErr: app.ErrTimeout},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupConfigEnvironment(t)
			reader := memory.NewResourceReader()
			command := NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
				return app.NewResourceService(reader, logger, progressReporter, app.WithFlakyDependency(memory.NewFlakyReader(app.FlakyFailures))), nil, nil
			}, testBuildInfo(), reader.Names())
			command.SetArgs(test.args)
			command.SetOut(io.Discard)
			command.SetErr(io.Discard)
			if err := command.Execute(); !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
