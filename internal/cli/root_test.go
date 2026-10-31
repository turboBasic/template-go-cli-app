package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
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
		{name: "json with forced color", args: []string{"--output", "json", "--color", "always", "resource", "list"}, bad: true},
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
	if err := os.Unsetenv("CLI_APP_OUTPUT"); err != nil {
		t.Fatal(err)
	}
	if err := os.Unsetenv("CLI_APP_QUIET"); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(configFile, []byte("log-level = 'info'\noutput = 'json'\nquiet = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := memory.NewResourceReader()
	var got config.Config
	factoryCalls := 0
	command := NewRootCommandWithFactory(func(_ context.Context, effective config.Config) (*app.ResourceService, error) {
		factoryCalls++
		got = effective
		return app.NewResourceService(reader), nil
	}, testBuildInfo(), reader.Names())
	command.SetArgs([]string{"--config", configFile, "--log-level", "debug", "resource", "list"})
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	want := config.Config{LogLevel: "debug", Output: "json", Color: "auto", Quiet: true}
	if got != want {
		t.Fatalf("factory config = %+v, want %+v", got, want)
	}
	if factoryCalls != 1 || !strings.Contains(output.String(), "example-1") {
		t.Fatalf("factory calls/output = %d/%q, want one call before resource execution", factoryCalls, output.String())
	}
}

func TestFactoryErrorPropagates(t *testing.T) {
	setupConfigEnvironment(t)
	wantErr := errors.New("factory failure")
	command := NewRootCommandWithFactory(func(context.Context, config.Config) (*app.ResourceService, error) {
		return nil, wantErr
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
			command := NewRootCommandWithFactory(func(context.Context, config.Config) (*app.ResourceService, error) {
				factoryCalls++
				return nil, errors.New("factory must not run")
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
	if err := os.WriteFile(firstConfig, []byte("log-level = 'info'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondConfig, []byte("log-level = 'error'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := memory.NewResourceReader()
	var configs []config.Config
	command := NewRootCommandWithFactory(func(_ context.Context, effective config.Config) (*app.ResourceService, error) {
		configs = append(configs, effective)
		return app.NewResourceService(reader), nil
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
	return NewRootCommand(app.NewResourceService(reader), testBuildInfo(), reader.Names())
}
