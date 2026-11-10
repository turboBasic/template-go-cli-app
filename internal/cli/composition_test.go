package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
)

type compositionConfig struct {
	First  compositionGroupSettings `koanf:"first"`
	Second compositionGroupSettings `koanf:"second"`
}

type compositionGroupSettings struct {
	Label   string            `koanf:"label"`
	Timeout *time.Duration    `koanf:"timeout"`
	Secret  compositionSecret `koanf:"secret"`
}

type compositionSecret struct {
	Reference string `koanf:"reference"`
}

type compositionState struct {
	setupCalls        map[string]int
	overrideCalls     map[string]int
	cleanupCalls      map[string]int
	services          map[string]string
	usedServices      map[string]string
	secretResolutions int
	operationTimeout  time.Duration
}

func compositionRegistration(state *compositionState) CommandRegistration {
	firstTimeout, secondTimeout := 100*time.Millisecond, 200*time.Millisecond
	defaults := compositionConfig{
		First:  compositionGroupSettings{Label: "first-default", Timeout: &firstTimeout},
		Second: compositionGroupSettings{Label: "second-default", Timeout: &secondTimeout},
	}
	var loaded compositionConfig
	return CommandRegistration{
		Settings: config.ApplicationSettings{
			Defaults: defaults,
			Validate: func(value any) error {
				settings, ok := value.(compositionConfig)
				if !ok {
					return fmt.Errorf("expected composition settings, got %T", value)
				}
				for name, group := range map[string]compositionGroupSettings{"first": settings.First, "second": settings.Second} {
					if group.Label == "" {
						return fmt.Errorf("%s.label is required", name)
					}
					if group.Timeout != nil && *group.Timeout <= 0 {
						return fmt.Errorf("%s.timeout must be positive", name)
					}
				}
				return nil
			},
		},
		PrepareApplication: func(value any) (any, error) {
			settings, ok := value.(compositionConfig)
			if !ok {
				return nil, fmt.Errorf("expected composition settings, got %T", value)
			}
			loaded = settings
			return settings, nil
		},
		RegisterCommands: func(root *cobra.Command, runtime *CommandRuntime) {
			registerGroup := func(name string, settings *compositionGroupSettings) {
				group := &cobra.Command{Use: name}
				runtime.SetCommandPolicy(group, CommandPolicy{
					Setup: SetupPolicy{Declared: true, Run: func(_ *cobra.Command, runtime *CommandRuntime) error {
						state.setupCalls[name]++
						state.services[name] = settings.Label
						runtime.RegisterCleanup(func(context.Context) error {
							state.cleanupCalls[name]++
							return nil
						})
						return nil
					}},
					Timeout: func(*cobra.Command) *time.Duration { return settings.Timeout },
				})
				group.AddCommand(&cobra.Command{Use: "run", RunE: func(*cobra.Command, []string) error {
					if runtime.Configuration.Application != loaded {
						return errors.New("command did not receive prepared application settings")
					}
					state.usedServices[name] = state.services[name]
					return nil
				}})
				override := &cobra.Command{Use: "override", RunE: func(*cobra.Command, []string) error {
					state.usedServices[name] = state.services[name]
					return nil
				}}
				runtime.SetCommandPolicy(override, CommandPolicy{Setup: SetupPolicy{Declared: true, Run: func(_ *cobra.Command, runtime *CommandRuntime) error {
					state.overrideCalls[name]++
					state.services[name] = name + "-override"
					runtime.RegisterCleanup(func(context.Context) error {
						state.cleanupCalls[name]++
						return nil
					})
					return nil
				}}})
				group.AddCommand(override)
				none := &cobra.Command{Use: "none", Run: func(*cobra.Command, []string) {}}
				runtime.SetCommandPolicy(none, CommandPolicy{Setup: SetupPolicy{Declared: true}})
				group.AddCommand(none)
				group.AddCommand(&cobra.Command{Use: "timeout", RunE: func(cmd *cobra.Command, _ []string) error {
					if deadline, ok := cmd.Context().Deadline(); ok {
						state.operationTimeout = time.Until(deadline)
					}
					<-cmd.Context().Done()
					return cmd.Context().Err()
				}})
				group.AddCommand(&cobra.Command{Use: "secret", Run: func(*cobra.Command, []string) {
					state.secretResolutions++
				}})
				root.AddCommand(group)
			}
			registerGroup("first", &loaded.First)
			registerGroup("second", &loaded.Second)
		},
	}
}

func newCompositionRoot(state *compositionState, args ...string) *cobra.Command {
	root := NewRootCommand(compositionRegistration(state), BuildInfo{})
	root.SetArgs(args)
	return root
}

func newCompositionState() *compositionState {
	return &compositionState{
		setupCalls: make(map[string]int), overrideCalls: make(map[string]int),
		cleanupCalls: make(map[string]int), services: make(map[string]string),
		usedServices: make(map[string]string),
	}
}

func TestMultipleCommandGroupsComposeIndependentSetup(t *testing.T) {
	setupConfigEnvironment(t)
	for _, test := range []struct {
		group string
		label string
	}{
		{group: "first", label: "first-default"},
		{group: "second", label: "second-default"},
	} {
		t.Run(test.group, func(t *testing.T) {
			state := newCompositionState()
			root := newCompositionRoot(state, test.group, "run")
			if err := Execute(context.Background(), root); err != nil {
				t.Fatal(err)
			}
			if state.setupCalls[test.group] != 1 || state.cleanupCalls[test.group] != 1 || state.usedServices[test.group] != test.label {
				t.Fatalf("selected group setup/cleanup/service = %d/%d/%q, want 1/1/%q", state.setupCalls[test.group], state.cleanupCalls[test.group], state.usedServices[test.group], test.label)
			}
			other := "first"
			if test.group == "first" {
				other = "second"
			}
			if state.setupCalls[other] != 0 {
				t.Fatalf("unselected group %q setup calls = %d, want zero", other, state.setupCalls[other])
			}
		})
	}
}

func TestMultipleCommandGroupPoliciesAndTypedTimeout(t *testing.T) {
	setupConfigEnvironment(t)
	for _, test := range []struct {
		name      string
		args      []string
		wantSetup int
		wantOver  int
		wantUsed  string
	}{
		{name: "group default", args: []string{"first", "run"}, wantSetup: 1, wantUsed: "first-default"},
		{name: "command override", args: []string{"first", "override"}, wantOver: 1, wantUsed: "first-override"},
		{name: "explicit no setup", args: []string{"first", "none"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := newCompositionState()
			if err := Execute(context.Background(), newCompositionRoot(state, test.args...)); err != nil {
				t.Fatal(err)
			}
			if state.setupCalls["first"] != test.wantSetup || state.overrideCalls["first"] != test.wantOver || state.usedServices["first"] != test.wantUsed {
				t.Fatalf("setup/override/service = %d/%d/%q, want %d/%d/%q", state.setupCalls["first"], state.overrideCalls["first"], state.usedServices["first"], test.wantSetup, test.wantOver, test.wantUsed)
			}
		})
	}

	for _, test := range []struct {
		group string
		want  time.Duration
	}{
		{group: "first", want: 100 * time.Millisecond},
		{group: "second", want: 200 * time.Millisecond},
	} {
		t.Run(test.group+" typed timeout", func(t *testing.T) {
			state := newCompositionState()
			err := Execute(context.Background(), newCompositionRoot(state, test.group, "timeout"))
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("Execute() error = %v, want deadline exceeded", err)
			}
			if remaining := state.operationTimeout; remaining < test.want-50*time.Millisecond || remaining > test.want {
				t.Fatalf("operation timeout remaining = %s, want near %s", remaining, test.want)
			}
		})
	}
}

func TestMultipleCommandGroupsValidateWholeConfigAndSkipInspectionSetup(t *testing.T) {
	setupConfigEnvironment(t)
	invalid := writeReplacementConfig(t, "[first]\nlabel = 'valid'\n[second]\nlabel = ''\n")
	state := newCompositionState()
	root := newCompositionRoot(state, "--config", invalid, "first", "run")
	if err := Execute(context.Background(), root); !errors.Is(err, app.ErrValidation) || !strings.Contains(err.Error(), "second.label") {
		t.Fatalf("Execute() error = %v, want validation of unselected second group", err)
	}
	if state.setupCalls["first"] != 0 || state.setupCalls["second"] != 0 {
		t.Fatalf("setup calls after whole-config failure = %v, want zero", state.setupCalls)
	}

	valid := writeReplacementConfig(t, "[first.secret]\nreference = 'op://vault/item/field'\n")
	for _, test := range []struct {
		name string
		args []string
	}{
		{name: "help", args: []string{"--config", filepath.Join(t.TempDir(), "missing.toml"), "--help"}},
		{name: "completion", args: []string{"--config", filepath.Join(t.TempDir(), "missing.toml"), "completion", "bash"}},
		{name: "completion request", args: []string{"--config", filepath.Join(t.TempDir(), "missing.toml"), "__complete", "first", ""}},
		{name: "completion request without descriptions", args: []string{"--config", filepath.Join(t.TempDir(), "missing.toml"), "__completeNoDesc", "first", ""}},
		{name: "config inspection", args: []string{"--config", valid, "config", "show"}},
		{name: "invalid config inspection", args: []string{"--config", invalid, "config", "show"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := newCompositionState()
			err := Execute(context.Background(), newCompositionRoot(state, test.args...))
			if strings.HasPrefix(test.name, "invalid") {
				if !errors.Is(err, app.ErrValidation) {
					t.Fatalf("Execute() error = %v, want validation error", err)
				}
			} else if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if state.setupCalls["first"] != 0 || state.setupCalls["second"] != 0 || state.secretResolutions != 0 {
				t.Fatalf("setup/secret resolution = %v/%d, want zero", state.setupCalls, state.secretResolutions)
			}
		})
	}
}

func TestNestedHelpAndCompletionNamesRunApplicationLifecycle(t *testing.T) {
	setupConfigEnvironment(t)
	for _, name := range []string{"help", "completion", "__complete", "__completeNoDesc", "__complete-extra"} {
		t.Run(name, func(t *testing.T) {
			state := newCompositionState()
			registration := compositionRegistration(state)
			register := registration.RegisterCommands
			registration.RegisterCommands = func(root *cobra.Command, runtime *CommandRuntime) {
				register(root, runtime)
				group, _, err := root.Find([]string{"first"})
				if err != nil {
					t.Fatal(err)
				}
				group.AddCommand(&cobra.Command{Use: name, RunE: func(cmd *cobra.Command, _ []string) error {
					if _, ok := cmd.Context().Deadline(); !ok {
						return errors.New("missing operation deadline")
					}
					if state.services["first"] != "first-default" {
						return errors.New("missing prepared group service")
					}
					return nil
				}})
			}
			root := NewRootCommand(registration, BuildInfo{})
			root.SetArgs([]string{"first", name})
			if err := Execute(context.Background(), root); err != nil {
				t.Fatal(err)
			}
			if state.setupCalls["first"] != 1 || state.cleanupCalls["first"] != 1 {
				t.Fatalf("setup/cleanup = %d/%d, want 1/1", state.setupCalls["first"], state.cleanupCalls["first"])
			}
		})
	}
}

func TestRootInfrastructureNamesAreReserved(t *testing.T) {
	setupConfigEnvironment(t)
	for _, name := range []string{"help", "completion", "config", "__complete", "__completeNoDesc"} {
		for _, alias := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/alias=%t", name, alias), func(t *testing.T) {
				registration := CommandRegistration{RegisterCommands: func(root *cobra.Command, _ *CommandRuntime) {
					command := &cobra.Command{Use: name, Run: func(*cobra.Command, []string) {}}
					if alias {
						command.Use = "application"
						command.Aliases = []string{name}
					}
					root.AddCommand(command, &cobra.Command{Use: "run", Run: func(*cobra.Command, []string) {}})
				}}
				root := NewRootCommand(registration, BuildInfo{})
				root.SetArgs([]string{"run"})
				if err := Execute(context.Background(), root); err == nil || !strings.Contains(err.Error(), "is reserved") {
					t.Fatalf("Execute() error = %v, want reserved root name error", err)
				}
			})
		}
	}
}
