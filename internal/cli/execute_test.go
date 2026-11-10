package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
)

func TestExecuteKeepsStage1RootFeatures(t *testing.T) {
	t.Run("completion is generated once", func(t *testing.T) {
		var want, got bytes.Buffer
		plain := newTestRootCommand(t)
		plain.SetOut(&want)
		plain.SetArgs([]string{"completion", "bash"})
		if err := plain.Execute(); err != nil {
			t.Fatal(err)
		}
		fanged := newTestRootCommand(t)
		fanged.SetOut(&got)
		fanged.SetArgs([]string{"completion", "bash"})
		if err := Execute(context.Background(), fanged); err != nil {
			t.Fatal(err)
		}
		if got.String() != want.String() || strings.Count(got.String(), "# bash completion V2 for cli-app") != 1 {
			t.Fatalf("completion bash through Execute differs from the Stage 1 script or repeats it")
		}
	})

	t.Run("man is an unknown command", func(t *testing.T) {
		command := newTestRootCommand(t)
		command.SetErr(&bytes.Buffer{})
		command.SetArgs([]string{"man"})
		if err := Execute(context.Background(), command); err == nil || !strings.Contains(err.Error(), `unknown command "man"`) {
			t.Fatalf("Execute(man) error = %v, want unknown command", err)
		}
	})

	t.Run("version keeps the Stage 1 template", func(t *testing.T) {
		var out bytes.Buffer
		command := newTestRootCommand(t)
		command.SetOut(&out)
		command.SetArgs([]string{"--version"})
		if err := Execute(context.Background(), command); err != nil {
			t.Fatal(err)
		}
		if want := "cli-app version 1.2.3 (commit abc123, built test-build)\n"; out.String() != want {
			t.Fatalf("--version = %q, want %q", out.String(), want)
		}
	})
}

func TestExecuteHelpAndCompletionSkipFactory(t *testing.T) {
	setupConfigEnvironment(t)
	for _, args := range [][]string{{"--help"}, {"help"}, {"completion", "bash"}, {"__complete", "demo", ""}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			factoryCalls := 0
			command := newTestDemoRoot(func(context.Context, config.Config, DemoConfig, *slog.Logger, app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
				factoryCalls++
				return nil, nil, errors.New("factory must not run")
			}, testBuildInfo(), nil)
			command.SetArgs(args)
			command.SetOut(&bytes.Buffer{})
			command.SetErr(&bytes.Buffer{})
			if err := Execute(context.Background(), command); err != nil {
				t.Fatalf("Execute(%v) error = %v", args, err)
			}
			if factoryCalls != 0 {
				t.Fatalf("factory calls = %d, want zero", factoryCalls)
			}
		})
	}
}

func TestExecuteReportsJoinedErrorOnce(t *testing.T) {
	cause := errors.New("interrupted by test")
	for _, test := range []struct {
		name    string
		run     func(*cobra.Command) error
		want    string
		wantErr error
	}{
		{"error and cause", func(*cobra.Command) error { return errors.New("operation failed") }, "operation failed\ninterrupted by test\n", cause},
		{"cause alone", func(*cobra.Command) error { return nil }, "interrupted by test\n", cause},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(cause)
			root := &cobra.Command{Use: "x", RunE: func(cmd *cobra.Command, _ []string) error { return test.run(cmd) }}
			var stderr bytes.Buffer
			root.SetErr(&stderr)
			root.SetArgs(nil)
			err := Execute(ctx, root)
			if !errors.Is(err, test.wantErr) || stderr.String() != test.want {
				t.Fatalf("Execute() = %v, stderr %q; want cause kept and stderr %q", err, stderr.String(), test.want)
			}
		})
	}
	t.Run("success prints nothing", func(t *testing.T) {
		root := &cobra.Command{Use: "x", RunE: func(*cobra.Command, []string) error { return nil }}
		var stderr bytes.Buffer
		root.SetErr(&stderr)
		root.SetArgs(nil)
		if err := Execute(context.Background(), root); err != nil || stderr.Len() != 0 {
			t.Fatalf("Execute() = %v, stderr %q; want nil and empty", err, stderr.String())
		}
	})
}

func TestGroupsRejectUnknownSubcommandsButBareGroupsShowHelp(t *testing.T) {
	setupConfigEnvironment(t)
	newCommand := func(args ...string) (*cobra.Command, *int) {
		calls := 0
		command := newTestDemoRoot(func(context.Context, config.Config, DemoConfig, *slog.Logger, app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
			calls++
			return nil, nil, errors.New("factory must not run")
		}, testBuildInfo(), nil)
		command.SetArgs(args)
		command.SetOut(&bytes.Buffer{})
		command.SetErr(&bytes.Buffer{})
		return command, &calls
	}

	for _, args := range [][]string{{"nope"}, {"demo", "nope"}, {"completion", "nope"}} {
		command, calls := newCommand(args...)
		if err := Execute(context.Background(), command); !errors.Is(err, app.ErrValidation) || !strings.Contains(err.Error(), `unknown command "nope"`) || *calls != 0 {
			t.Fatalf("Execute(%v) = %v, factory calls %d; want a validation error and no factory call", args, err, *calls)
		}
	}
	for _, args := range [][]string{nil, {"--output", "xml"}, {"demo"}, {"completion"}} {
		command, calls := newCommand(args...)
		var out bytes.Buffer
		command.SetOut(&out)
		if err := Execute(context.Background(), command); err != nil || out.Len() == 0 || *calls != 0 {
			t.Fatalf("Execute(%v) = %v, output %d bytes, factory calls %d; want help, no error, no factory call", args, err, out.Len(), *calls)
		}
	}
}

func TestApplicationGroupLifecycleThroughExecute(t *testing.T) {
	setupConfigEnvironment(t)
	type settings struct {
		Label string `koanf:"label"`
	}
	configReads, setupCalls, operationCalls := 0, 0, 0
	registration := CommandRegistration{
		Settings: config.ApplicationSettings{Defaults: settings{Label: "loaded"}},
		FlagValues: func(*cobra.Command) map[string]config.FlagValue {
			configReads++
			return nil
		},
		RegisterCommands: func(root *cobra.Command, runtime *CommandRuntime) {
			group := &cobra.Command{Use: "servers"}
			runtime.SetCommandPolicy(group, CommandPolicy{Setup: SetupPolicy{Declared: true, Run: func(*cobra.Command, *CommandRuntime) error {
				setupCalls++
				return nil
			}}})
			group.AddCommand(&cobra.Command{Use: "list", RunE: func(*cobra.Command, []string) error {
				if got := runtime.Configuration.Application.(settings).Label; got != "loaded" {
					return fmt.Errorf("application label = %q, want loaded", got)
				}
				operationCalls++
				return nil
			}})
			root.AddCommand(group)
		},
	}
	missing := filepath.Join(t.TempDir(), "missing.toml")
	for _, args := range [][]string{{"--config", missing, "servers"}, {"--config", missing, "servers", "--help"}} {
		var out bytes.Buffer
		root := NewRootCommand(registration, BuildInfo{})
		root.SetArgs(args)
		root.SetOut(&out)
		root.SetErr(&bytes.Buffer{})
		if err := Execute(context.Background(), root); err != nil || out.Len() == 0 || configReads != 0 || setupCalls != 0 {
			t.Fatalf("Execute(%v) = %v, output %d bytes, config reads/setup %d/%d; want help without loading config or setup", args, err, out.Len(), configReads, setupCalls)
		}
	}
	root := NewRootCommand(registration, BuildInfo{})
	root.SetArgs([]string{"servers", "lst"})
	root.SetErr(&bytes.Buffer{})
	if err := Execute(context.Background(), root); !errors.Is(err, app.ErrValidation) || !strings.Contains(err.Error(), `unknown command "lst"`) || configReads != 0 || setupCalls != 0 {
		t.Fatalf("Execute(servers lst) = %v, config reads/setup %d/%d; want validation error and no configuration access or setup", err, configReads, setupCalls)
	}
	root = NewRootCommand(registration, BuildInfo{})
	root.SetArgs([]string{"servers", "list"})
	root.SetErr(&bytes.Buffer{})
	if err := Execute(context.Background(), root); err != nil || configReads != 1 || setupCalls != 1 || operationCalls != 1 {
		t.Fatalf("Execute(servers list) = %v, config reads/setup/operation %d/%d/%d; want 1/1/1", err, configReads, setupCalls, operationCalls)
	}
	for _, args := range [][]string{{"config", "show"}, {"--help"}, {"completion", "bash"}} {
		root = NewRootCommand(registration, BuildInfo{})
		root.SetArgs(args)
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		if err := Execute(context.Background(), root); err != nil {
			t.Fatalf("Execute(%v) = %v", args, err)
		}
	}
}

var ansiSequence = regexp.MustCompile("\x1b\\[[0-9;]*m")

func runFailing(t *testing.T, failure error) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "x", RunE: func(*cobra.Command, []string) error { return failure }}
	var stderr bytes.Buffer
	root.SetErr(&stderr)
	root.SetArgs(nil)
	err := Execute(context.Background(), root)
	return stderr.String(), err
}

func TestErrorHandlerPlainWhenColorIsOff(t *testing.T) {
	failure := app.Classify(app.ErrValidation, errors.New("bad input"))
	for name, env := range map[string]map[string]string{
		"redirected": {"TERM": "xterm-256color"},
		"TERM=dumb":  {"TERM": "dumb"},
	} {
		t.Run(name, func(t *testing.T) {
			for _, key := range []string{"TERM", "CLICOLOR_FORCE", "NO_COLOR"} {
				t.Setenv(key, env[key])
			}
			stderr, err := runFailing(t, failure)
			if !errors.Is(err, app.ErrValidation) || stderr != failure.Error()+"\n" {
				t.Fatalf("stderr = %q, error = %v; want exactly %q", stderr, err, failure.Error()+"\n")
			}
			if strings.Contains(stderr, "38;") || strings.Contains(stderr, "48;") {
				t.Fatalf("stderr = %q, want no color sequences", stderr)
			}
		})
	}
}

func TestErrorHandlerStyledWhenColorIsForced(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("CLICOLOR_FORCE", "1")
	t.Setenv("NO_COLOR", "1") // like Fang's help and gh, forcing wins

	stderr, _ := runFailing(t, app.Classify(app.ErrValidation, errors.New("bad input")))
	plain := ansiSequence.ReplaceAllString(stderr, "")
	for _, want := range []string{"ERROR", "bad input", "Try --help for usage."} {
		if !strings.Contains(plain, want) {
			t.Errorf("validation error output lacks %q:\n%s", want, plain)
		}
	}
	if stderr == plain {
		t.Errorf("validation error output carries no styling: %q", stderr)
	}

	stderr, _ = runFailing(t, context.DeadlineExceeded)
	plain = ansiSequence.ReplaceAllString(stderr, "")
	if !strings.Contains(plain, "ERROR") || !strings.Contains(strings.ToLower(plain), "deadline exceeded") || strings.Contains(plain, "--help") {
		t.Errorf("deadline output = %q, want the ERROR block without a usage hint", plain)
	}
}

func TestHelpIsStyledOnlyWhenColorIsForced(t *testing.T) {
	for _, mode := range []struct{ name, force string }{{"plain", ""}, {"styled", "1"}} {
		t.Run(mode.name, func(t *testing.T) {
			t.Setenv("TERM", "xterm-256color")
			t.Setenv("COLORTERM", "")
			t.Setenv("NO_COLOR", "")
			t.Setenv("CLICOLOR", "")
			t.Setenv("CLICOLOR_FORCE", mode.force)
			for _, test := range []struct {
				args []string
				want []string
			}{
				{[]string{"--help"}, []string{"demo", "completion", "--output"}},
				{[]string{"demo", "--help"}, []string{"structured-output", "not-found", "secret-credential"}},
				{[]string{"demo", "not-found", "--help"}, []string{"not-found", "--output"}},
				{[]string{"help", "demo"}, []string{"structured-output", "not-found", "secret-credential"}},
			} {
				t.Run(strings.Join(test.args, "_"), func(t *testing.T) {
					var stdout, stderr bytes.Buffer
					command := newTestRootCommand(t)
					command.SetOut(&stdout)
					command.SetErr(&stderr)
					command.SetArgs(test.args)
					if err := Execute(context.Background(), command); err != nil || stderr.Len() != 0 {
						t.Fatalf("Execute(%v) = %v, stderr %q; want success and empty stderr", test.args, err, stderr.String())
					}
					styled := strings.Contains(stdout.String(), "\x1b[")
					if styled != (mode.force == "1") {
						t.Errorf("help styled = %v in %s mode:\n%q", styled, mode.name, stdout.String())
					}
					plain := ansiSequence.ReplaceAllString(stdout.String(), "")
					for _, want := range test.want {
						if !strings.Contains(plain, want) {
							t.Errorf("help lacks %q:\n%s", want, plain)
						}
					}
				})
			}
		})
	}
}
