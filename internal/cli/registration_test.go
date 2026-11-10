package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
	"github.com/turboBasic/template-go-cli-app/internal/output"
)

func TestCommandPolicyResolution(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	group := &cobra.Command{Use: "group"}
	child := &cobra.Command{Use: "child"}
	override := &cobra.Command{Use: "override"}
	noSetup := &cobra.Command{Use: "no-setup"}
	root.AddCommand(group)
	group.AddCommand(child, override, noSetup)
	groupTimeout := time.Second
	childTimeout := 2 * time.Second
	runtime := &CommandRuntime{}
	runtime.SetCommandPolicy(group, CommandPolicy{
		Setup: SetupPolicy{Declared: true, Run: func(*cobra.Command, *CommandRuntime) error { return nil }},
		Timeout: func(command *cobra.Command) *time.Duration {
			if command.Parent() == group {
				return &groupTimeout
			}
			return nil
		},
	})
	runtime.SetCommandPolicy(override, CommandPolicy{
		Setup: SetupPolicy{Declared: true, Run: func(*cobra.Command, *CommandRuntime) error { return nil }},
		Timeout: func(command *cobra.Command) *time.Duration {
			if command == override {
				return &childTimeout
			}
			return nil
		},
	})
	runtime.SetCommandPolicy(noSetup, CommandPolicy{Setup: SetupPolicy{Declared: true}})

	for _, test := range []struct {
		name        string
		command     *cobra.Command
		wantSetup   bool
		wantTimeout time.Duration
	}{
		{name: "group default", command: child, wantSetup: true, wantTimeout: groupTimeout},
		{name: "child override", command: override, wantSetup: true, wantTimeout: childTimeout},
		{name: "explicit no setup", command: noSetup, wantTimeout: groupTimeout},
	} {
		t.Run(test.name, func(t *testing.T) {
			policy := runtime.selectedPolicy(test.command)
			if got := policy.Setup.Run != nil; got != test.wantSetup {
				t.Fatalf("setup present = %t, want %t", got, test.wantSetup)
			}
			if got := *policy.Timeout(test.command); got != test.wantTimeout {
				t.Fatalf("timeout = %s, want %s", got, test.wantTimeout)
			}
		})
	}
}

func TestPrepareApplicationRejectsTypeMismatch(t *testing.T) {
	setupConfigEnvironment(t)
	want := errors.New("expected replacement settings")
	calls := 0
	registration := CommandRegistration{
		Settings: config.ApplicationSettings{Defaults: DemoConfig{}},
		PrepareApplication: func(value any) (any, error) {
			calls++
			if _, ok := value.(replacementSettings); !ok {
				return nil, want
			}
			return value, nil
		},
		RegisterCommands: func(root *cobra.Command, _ *CommandRuntime) {
			root.AddCommand(&cobra.Command{Use: "run", Run: func(*cobra.Command, []string) {}})
		},
	}
	root := NewRootCommand(registration, BuildInfo{})
	root.SetArgs([]string{"run"})
	err := root.Execute()
	if !errors.Is(err, want) || !strings.Contains(err.Error(), "prepare application settings") {
		t.Fatalf("Execute() error = %v, want clear type mismatch", err)
	}
	if calls != 1 {
		t.Fatalf("application conversion calls = %d, want one", calls)
	}
}

func TestPreparedApplicationValueFlowsToRuntimeAndTimeout(t *testing.T) {
	setupConfigEnvironment(t)
	checked := replacementSettings{Label: "checked"}
	seenByTimeout := false
	seenByHandler := false
	registration := CommandRegistration{
		Settings: config.ApplicationSettings{Defaults: replacementSettings{Label: "loaded"}},
		PrepareApplication: func(value any) (any, error) {
			if _, ok := value.(replacementSettings); !ok {
				return nil, fmt.Errorf("unexpected settings type %T", value)
			}
			return checked, nil
		},
		OperationTimeout: func(_ *cobra.Command, value any) *time.Duration {
			seenByTimeout = value == checked
			return nil
		},
		RegisterCommands: func(root *cobra.Command, runtime *CommandRuntime) {
			root.AddCommand(&cobra.Command{Use: "run", Run: func(*cobra.Command, []string) {
				seenByHandler = runtime.Configuration.Application == checked
			}})
		},
	}
	root := NewRootCommand(registration, BuildInfo{})
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !seenByTimeout || !seenByHandler {
		t.Fatalf("prepared value observed by timeout/handler = %t/%t, want true/true", seenByTimeout, seenByHandler)
	}
}

type replacementResult struct {
	Count int `json:"count"`
}

type replacementService struct{}

func (replacementService) Result() replacementResult { return replacementResult{Count: 4} }

type replacementSettings struct {
	Label string `koanf:"label"`
}

func TestReplacementRegistrationUsesSharedTextAndJSONRenderer(t *testing.T) {
	setupConfigEnvironment(t)
	for _, test := range []struct {
		name, format, want string
		wantTextCalls      int
	}{
		{name: "text callback", format: "text", want: "count: 4\n", wantTextCalls: 1},
		{name: "normal JSON", format: "json", want: "{\"count\":4}\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			textCalls := 0
			registration := CommandRegistration{
				Settings: config.ApplicationSettings{Defaults: replacementSettings{Label: "sample"}},
				RegisterCommands: func(root *cobra.Command, runtime *CommandRuntime) {
					root.AddCommand(&cobra.Command{Use: "replacement", RunE: func(_ *cobra.Command, _ []string) error {
						service := replacementService{}
						result := service.Result()
						if runtime.Configuration.Application.(replacementSettings).Label != "sample" || runtime.Logger == nil || runtime.Progress == nil {
							return fmt.Errorf("shared runtime or registered config was not populated")
						}
						runtime.Logger.Debug("replacement command")
						runtime.Progress.Report(app.ProgressEvent{Phase: "complete", Message: "replacement ready"})
						return runtime.Renderer.RenderResult(result, func(w io.Writer, options output.Options) error {
							textCalls++
							if options.Format != output.FormatText {
								return fmt.Errorf("text options format = %q", options.Format)
							}
							_, err := fmt.Fprintf(w, "count: %d\n", result.Count)
							return err
						})
					}})
				},
			}
			root := NewRootCommand(registration, BuildInfo{})
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs([]string{"--log-level", "debug", "--output", test.format, "replacement"})
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if stdout.String() != test.want {
				t.Fatalf("stdout = %q, want %q", stdout.String(), test.want)
			}
			if !strings.Contains(stderr.String(), "msg=\"replacement command\"") || !strings.Contains(stderr.String(), "replacement ready") {
				t.Fatalf("stderr = %q, want replacement log and progress", stderr.String())
			}
			if textCalls != test.wantTextCalls {
				t.Fatalf("text callback calls = %d, want %d", textCalls, test.wantTextCalls)
			}
		})
	}
}
