package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
)

type replacementAppConfig struct {
	Worker replacementWorkerConfig `koanf:"worker"`
}

type replacementWorkerConfig struct {
	Label    string         `koanf:"label"`
	Mode     string         `koanf:"mode"`
	Timeout  *time.Duration `koanf:"timeout"`
	APIToken string         `koanf:"api_token"`
}

type lifecycleState struct {
	setupCalls        int
	cleanupCalls      int
	secretResolutions int
	setupErr          error
	operationErr      error
	cleanupErr        error
	cleanupContextErr error
	cleanupDeadline   time.Time
	cancel            context.CancelFunc
	loaded            config.LoadResult
	operationName     string
	operationTimeout  time.Duration
}

type commandPolicyState struct {
	setupErr, cleanupErr error
	blockSetup           bool
	setupCalls           int
	handlerCalls         int
	cleanupCalls         int
	cleanupContextErr    error
	setupDeadline        time.Duration
}

func commandPolicyRegistration(state *commandPolicyState) CommandRegistration {
	return CommandRegistration{
		RegisterCommands: func(root *cobra.Command, runtime *CommandRuntime) {
			command := &cobra.Command{Use: "run", RunE: func(*cobra.Command, []string) error {
				state.handlerCalls++
				return nil
			}}
			runtime.SetCommandPolicy(command, CommandPolicy{
				Setup: SetupPolicy{Declared: true, Run: func(cmd *cobra.Command, runtime *CommandRuntime) error {
					state.setupCalls++
					if runtime.Logger == nil || runtime.Progress == nil {
						return errors.New("shared runtime is not initialized")
					}
					if deadline, ok := cmd.Context().Deadline(); ok {
						state.setupDeadline = time.Until(deadline)
					}
					runtime.RegisterCleanup(func(ctx context.Context) error {
						state.cleanupCalls++
						state.cleanupContextErr = ctx.Err()
						return state.cleanupErr
					})
					if state.blockSetup {
						<-cmd.Context().Done()
						return cmd.Context().Err()
					}
					return state.setupErr
				}},
				Timeout: func(*cobra.Command) *time.Duration {
					timeout := 2 * time.Second
					return &timeout
				},
			})
			root.AddCommand(command)
		},
	}
}

func TestCommandPolicySetupRunsInsideOperationLifecycle(t *testing.T) {
	setupConfigEnvironment(t)
	setupErr, cleanupErr := errors.New("partial setup failed"), errors.New("cleanup failed")
	for _, test := range []struct {
		name             string
		args             []string
		setupErr         error
		cleanupErr       error
		blockSetup       bool
		wantSetupErr     error
		wantCleanupErr   error
		wantSetupCalls   int
		wantCleanupCalls int
		wantHandlerCalls int
	}{
		{name: "success", args: []string{"run"}, wantSetupCalls: 1, wantCleanupCalls: 1, wantHandlerCalls: 1},
		{name: "partial setup failure", args: []string{"run"}, setupErr: setupErr, cleanupErr: cleanupErr, wantSetupErr: setupErr, wantCleanupErr: cleanupErr, wantSetupCalls: 1, wantCleanupCalls: 1},
		{name: "setup deadline", args: []string{"--timeout", "30ms", "run"}, blockSetup: true, wantSetupErr: context.DeadlineExceeded, wantSetupCalls: 1, wantCleanupCalls: 1},
		{name: "help", args: []string{"--help"}},
		{name: "completion", args: []string{"completion", "bash"}},
		{name: "config inspection", args: []string{"config", "show"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := &commandPolicyState{
				setupErr: test.setupErr, cleanupErr: test.cleanupErr, blockSetup: test.blockSetup,
			}
			root := NewRootCommand(commandPolicyRegistration(state), BuildInfo{})
			root.SetArgs(test.args)
			root.SetErr(io.Discard)
			err := Execute(context.Background(), root)
			if test.wantSetupErr != nil && !errors.Is(err, test.wantSetupErr) {
				t.Errorf("Execute() error = %v, want setup error %v", err, test.wantSetupErr)
			}
			if test.wantCleanupErr != nil && !errors.Is(err, test.wantCleanupErr) {
				t.Errorf("Execute() error = %v, want cleanup error %v", err, test.wantCleanupErr)
			}
			if test.wantSetupErr == nil && test.wantCleanupErr == nil && err != nil {
				t.Fatalf("Execute() error = %v, want nil", err)
			}
			if state.setupCalls != test.wantSetupCalls || state.cleanupCalls != test.wantCleanupCalls || state.handlerCalls != test.wantHandlerCalls {
				t.Fatalf("setup/cleanup/handler calls = %d/%d/%d, want %d/%d/%d", state.setupCalls, state.cleanupCalls, state.handlerCalls, test.wantSetupCalls, test.wantCleanupCalls, test.wantHandlerCalls)
			}
			if state.cleanupContextErr != nil {
				t.Fatalf("cleanup context error = %v, want live bounded context", state.cleanupContextErr)
			}
			if test.blockSetup && (state.setupDeadline <= 0 || state.setupDeadline > time.Second) {
				t.Fatalf("setup deadline remaining = %s, want explicit 30ms override", state.setupDeadline)
			}
		})
	}
}

type lifecycleReplacementService struct{ state *lifecycleState }

func (s lifecycleReplacementService) run(ctx context.Context, runtime *CommandRuntime, name string) error {
	s.state.operationName = name
	s.state.loaded = runtime.Configuration
	if deadline, ok := ctx.Deadline(); ok {
		s.state.operationTimeout = time.Until(deadline)
	}
	switch name {
	case "wait":
		<-ctx.Done()
		return ctx.Err()
	case "cancel":
		s.state.cancel()
		<-ctx.Done()
		return ctx.Err()
	case "failure":
		return s.state.operationErr
	case "resolve":
		s.state.secretResolutions++
	}
	return nil
}

type replacementServiceGroup struct {
	runtime     *CommandRuntime
	state       *lifecycleState
	runID       uint64
	initialized bool
	service     lifecycleReplacementService
	err         error
}

func (g *replacementServiceGroup) get(ctx context.Context) (lifecycleReplacementService, error) {
	if g.runID != g.runtime.RunID {
		g.runID, g.initialized, g.service, g.err = g.runtime.RunID, false, lifecycleReplacementService{}, nil
	}
	if !g.initialized {
		g.initialized = true
		g.state.setupCalls++
		cleanup := func(ctx context.Context) error {
			g.state.cleanupCalls++
			g.state.cleanupContextErr = ctx.Err()
			g.state.cleanupDeadline, _ = ctx.Deadline()
			return g.state.cleanupErr
		}
		if g.state.setupErr != nil {
			g.runtime.RegisterCleanup(cleanup)
			g.err = fmt.Errorf("replacement setup: %w", g.state.setupErr)
		} else {
			g.service = lifecycleReplacementService{state: g.state}
			g.runtime.RegisterCleanup(cleanup)
		}
	}
	return g.service, g.err
}

func replacementRegistration(state *lifecycleState) CommandRegistration {
	timeout := 5 * time.Second
	return CommandRegistration{
		Settings: config.ApplicationSettings{
			Defaults: replacementAppConfig{Worker: replacementWorkerConfig{Label: "default", Mode: "safe", Timeout: &timeout}},
			Validate: func(value any) error {
				settings := value.(replacementAppConfig)
				if settings.Worker.Label == "" {
					return errors.New("worker.label is required")
				}
				if settings.Worker.Timeout != nil && *settings.Worker.Timeout <= 0 {
					return errors.New("worker.timeout must be positive")
				}
				return nil
			},
		},
		FlagValues: func(cmd *cobra.Command) map[string]config.FlagValue {
			flag := cmd.Flags().Lookup("label")
			if flag == nil || !flag.Changed {
				return nil
			}
			value, _ := cmd.Flags().GetString("label")
			return map[string]config.FlagValue{"worker-label": {Value: value, Changed: true, Source: "--label"}}
		},
		OperationTimeout: func(cmd *cobra.Command, value any) *time.Duration {
			if cmd.Name() != "wait" && cmd.Name() != "cancel" {
				return nil
			}
			return value.(replacementAppConfig).Worker.Timeout
		},
		RegisterCommands: func(root *cobra.Command, runtime *CommandRuntime) {
			group := &replacementServiceGroup{runtime: runtime, state: state}
			for _, name := range []string{"first", "second", "failure", "wait", "cancel", "resolve"} {
				commandName := name
				command := &cobra.Command{Use: commandName, RunE: func(cmd *cobra.Command, _ []string) error {
					service, err := group.get(cmd.Context())
					if err != nil {
						return err
					}
					return service.run(cmd.Context(), runtime, commandName)
				}}
				command.Flags().String("label", "", "replacement label")
				root.AddCommand(command)
			}
		},
	}
}

func TestReplacementGroupSharesLazyDependenciesAcrossCommandsAndRuns(t *testing.T) {
	setupConfigEnvironment(t)
	state := &lifecycleState{}
	root := NewRootCommand(replacementRegistration(state), BuildInfo{})
	if state.setupCalls != 0 {
		t.Fatalf("setup calls during registration = %d, want zero", state.setupCalls)
	}
	for _, name := range []string{"first", "second"} {
		root.SetArgs([]string{name})
		if err := Execute(context.Background(), root); err != nil {
			t.Fatal(err)
		}
	}
	if state.setupCalls != 2 || state.cleanupCalls != 2 {
		t.Fatalf("setup/cleanup calls = %d/%d, want one of each per selected command run", state.setupCalls, state.cleanupCalls)
	}
}

func TestReplacementLifecycleCleansOnceWithFreshBoundedContext(t *testing.T) {
	setupConfigEnvironment(t)
	setupErr, operationErr, cleanupErr := errors.New("setup failed"), errors.New("operation failed"), errors.New("cleanup failed")
	for _, test := range []struct {
		name                               string
		args                               []string
		setupErr, operationErr, cleanupErr error
		wantErrors                         []error
	}{
		{name: "success", args: []string{"first"}},
		{name: "command failure and cleanup failure", args: []string{"failure"}, operationErr: operationErr, cleanupErr: cleanupErr, wantErrors: []error{operationErr, cleanupErr}},
		{name: "setup failure still cleans partial dependencies", args: []string{"first"}, setupErr: setupErr, cleanupErr: cleanupErr, wantErrors: []error{setupErr, cleanupErr}},
		{name: "timeout", args: []string{"wait"}, wantErrors: []error{context.DeadlineExceeded}},
		{name: "cancellation", args: []string{"cancel"}, wantErrors: []error{context.Canceled}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := &lifecycleState{setupErr: test.setupErr, operationErr: test.operationErr, cleanupErr: test.cleanupErr}
			root := NewRootCommand(replacementRegistration(state), BuildInfo{})
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			args := []string{"--config", writeReplacementConfig(t, "[timeouts]\ncleanup = '500ms'\n[worker]\ntimeout = '20ms'\n")}
			args = append(args, test.args...)
			root.SetArgs(args)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			state.cancel = cancel
			err := Execute(ctx, root)
			for _, want := range test.wantErrors {
				if !errors.Is(err, want) {
					t.Errorf("Execute() error = %v, want %v", err, want)
				}
			}
			if len(test.wantErrors) == 0 && err != nil {
				t.Fatalf("Execute() error = %v, want nil", err)
			}
			if state.cleanupCalls != 1 {
				t.Fatalf("cleanup calls = %d, want exactly one", state.cleanupCalls)
			}
			if state.cleanupContextErr != nil || state.cleanupDeadline.IsZero() || time.Until(state.cleanupDeadline) <= 0 || time.Until(state.cleanupDeadline) > 500*time.Millisecond {
				t.Fatalf("cleanup context error/deadline = %v/%v, want live context bounded to 500ms", state.cleanupContextErr, state.cleanupDeadline)
			}
		})
	}
}

func TestReplacementSettingsFlagsPrecedenceProvenanceAndTimeoutOverride(t *testing.T) {
	setupConfigEnvironment(t)
	path := writeReplacementConfig(t, "[worker]\nlabel = 'file'\nmode = 'file-mode'\ntimeout = '4s'\napi_token = 'secret-token'\n")
	t.Setenv("CLI_APP_WORKER_LABEL", "environment")
	state := &lifecycleState{}
	root := NewRootCommand(replacementRegistration(state), BuildInfo{})
	root.SetArgs([]string{"--config", path, "second", "--label", "flag"})
	if err := Execute(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	settings := state.loaded.Application.(replacementAppConfig)
	if settings.Worker.Label != "flag" || settings.Worker.Mode != "file-mode" || settings.Worker.Timeout == nil || *settings.Worker.Timeout != 4*time.Second {
		t.Fatalf("effective settings = %+v, want flag/file/default precedence", settings)
	}
	for key, want := range map[string]config.Source{
		"worker.label":   {Kind: config.KindFlag, Origin: "--label"},
		"worker.mode":    {Kind: config.KindProjectFile, Origin: path},
		"worker.timeout": {Kind: config.KindProjectFile, Origin: path},
	} {
		if state.loaded.Sources[key] != want {
			t.Errorf("source %s = %+v, want %+v", key, state.loaded.Sources[key], want)
		}
	}

	state = &lifecycleState{}
	root = NewRootCommand(replacementRegistration(state), BuildInfo{})
	root.SetArgs([]string{"--config", path, "--timeout", "20ms", "wait"})
	if err := Execute(context.Background(), root); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout override error = %v, want deadline exceeded", err)
	}
	if state.cleanupCalls != 1 {
		t.Fatalf("override cleanup calls = %d, want one", state.cleanupCalls)
	}
	if state.operationTimeout <= 0 || state.operationTimeout > 200*time.Millisecond {
		t.Fatalf("operation timeout = %s, want --timeout override near 20ms instead of app's 4s", state.operationTimeout)
	}
	if state.operationName != "wait" {
		t.Fatalf("operation = %q, want wait", state.operationName)
	}
}

func TestReplacementValidationAndInspectionDoNotInitializeSecrets(t *testing.T) {
	setupConfigEnvironment(t)
	for _, test := range []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "invalid registered settings", args: []string{"first"}, wantErr: true},
		{name: "config inspection", args: []string{"config", "show", "worker.api_token"}},
		{name: "root help", args: []string{"--help"}},
		{name: "subcommand help", args: []string{"first", "--help"}},
		{name: "help topic", args: []string{"help", "first"}},
		{name: "completion", args: []string{"completion", "bash"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "invalid registered settings" {
				t.Setenv("CLI_APP_WORKER_LABEL", "")
			}
			state := &lifecycleState{}
			path := writeReplacementConfig(t, "[worker]\napi_token = 'secret-token'\n")
			root := NewRootCommand(replacementRegistration(state), BuildInfo{})
			var stdout bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&bytes.Buffer{})
			args := append([]string{"--config", path}, test.args...)
			root.SetArgs(args)
			err := Execute(context.Background(), root)
			if test.wantErr && !errors.Is(err, app.ErrValidation) {
				t.Fatalf("Execute() error = %v, want validation error", err)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if state.setupCalls != 0 || state.secretResolutions != 0 {
				t.Fatalf("setup/secret resolutions = %d/%d, want zero", state.setupCalls, state.secretResolutions)
			}
			if test.name == "config inspection" && (!strings.Contains(stdout.String(), "[REDACTED]") || strings.Contains(stdout.String(), "secret-token")) {
				t.Fatalf("config show output = %q, want masked token", stdout.String())
			}
		})
	}
}

func TestSharedOnlyRootRejectsRemovedDemoAndUnknownAppKeys(t *testing.T) {
	setupConfigEnvironment(t)
	previous, wasSet := os.LookupEnv("CLI_APP_OUTPUT")
	if err := os.Unsetenv("CLI_APP_OUTPUT"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if wasSet {
			_ = os.Setenv("CLI_APP_OUTPUT", previous)
		}
	})
	sharedPath := writeReplacementConfig(t, "output = 'json'\n")
	root := NewRootCommand(CommandRegistration{}, BuildInfo{})
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"--config", sharedPath, "config", "show", "output"})
	if err := Execute(context.Background(), root); err != nil {
		t.Fatalf("shared-only config show: %v", err)
	}
	if !strings.Contains(stdout.String(), "json") {
		t.Fatalf("shared output = %q, want json setting", stdout.String())
	}
	for _, data := range []string{
		"[demo.secret_credential]\nreference = 'op://vault/item/field'\n",
		"[unknown_app]\nsetting = 'value'\n",
	} {
		path := writeReplacementConfig(t, data)
		root := NewRootCommand(CommandRegistration{}, BuildInfo{})
		root.SetArgs([]string{"--config", path, "config", "show"})
		if err := Execute(context.Background(), root); !errors.Is(err, app.ErrValidation) || !strings.Contains(err.Error(), "unknown key") {
			t.Errorf("shared-only load error = %v, want unknown-key validation", err)
		}
	}
}

func writeReplacementConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
