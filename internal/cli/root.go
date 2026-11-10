package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
	"github.com/turboBasic/template-go-cli-app/internal/logging"
	"github.com/turboBasic/template-go-cli-app/internal/output"
)

const configID = "cli-app"

// CommandRegistration supplies application commands, settings, and their config mappings.
type CommandRegistration struct {
	Settings         config.ApplicationSettings
	RegisterCommands func(*cobra.Command, *CommandRuntime)
	FlagValues       func(*cobra.Command) map[string]config.FlagValue
	OperationTimeout func(*cobra.Command, any) *time.Duration
	// PrepareApplication checks the loaded application value and returns the value closures use.
	PrepareApplication func(any) (any, error)
}

// SetupPolicy declares whether a command supplies a setup policy. A zero value inherits its parent.
type SetupPolicy struct {
	Declared bool
	Run      func(*cobra.Command, *CommandRuntime) error
}

// CommandPolicy associates setup and timeout behavior with a Cobra command.
type CommandPolicy struct {
	Setup   SetupPolicy
	Timeout func(*cobra.Command) *time.Duration
}

// CommandRuntime exposes shared facilities to commands after a selected command initializes them.
type CommandRuntime struct {
	Configuration   config.LoadResult
	Renderer        output.Renderer
	Logger          *slog.Logger
	Progress        app.ProgressReporter
	policies        map[*cobra.Command]CommandPolicy
	registerCleanup func(func(context.Context) error)
}

// SetCommandPolicy associates policy with the command object created during registration.
func (r *CommandRuntime) SetCommandPolicy(command *cobra.Command, policy CommandPolicy) {
	if r.policies == nil {
		r.policies = make(map[*cobra.Command]CommandPolicy)
	}
	r.policies[command] = policy
}

func (r *CommandRuntime) selectedPolicy(command *cobra.Command) CommandPolicy {
	var selected CommandPolicy
	for current := command; current != nil; current = current.Parent() {
		policy, ok := r.policies[current]
		if !ok {
			continue
		}
		if !selected.Setup.Declared && policy.Setup.Declared {
			selected.Setup = policy.Setup
		}
		if selected.Timeout == nil && policy.Timeout != nil {
			selected.Timeout = policy.Timeout
		}
	}
	return selected
}

// RegisterCleanup adds command-owned dependency cleanup to the root lifecycle.
func (r *CommandRuntime) RegisterCleanup(cleanup func(context.Context) error) {
	if cleanup != nil && r.registerCleanup != nil {
		r.registerCleanup(cleanup)
	}
}

// BuildInfo is the version metadata embedded in the executable.
type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

// NewRootCommand builds one standalone execution's static command tree and shared runtime lifecycle.
// Create a fresh registration and root for each execution. Application groups with children are
// help-only; put application operations on leaf commands. Application commands must not define
// persistent pre-run hooks; use setup policies instead.
func NewRootCommand(registration CommandRegistration, build BuildInfo) *cobra.Command {
	root := &cobra.Command{
		Use:     "cli-app",
		Short:   "Manage simulated infrastructure resources",
		Version: build.Version,
		Annotations: map[string]string{
			"commit":    build.Commit,
			"buildDate": build.BuildDate,
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rejectUnknownSubcommands(root)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return app.Classify(app.ErrValidation, err)
	})
	root.SetVersionTemplate(
		"{{printf \"%s version %s (commit %s, built %s)\\n\" .Name .Version (index .Annotations \"commit\") (index .Annotations \"buildDate\")}}",
	)
	var configPath, logLevel, outputFormat, color string
	var timeoutOverride time.Duration
	var quiet bool
	flags := root.PersistentFlags()
	flags.StringVar(&configPath, "config", "", "configuration file path")
	flags.StringVar(&logLevel, "log-level", "warn", "logging level (debug, info, warn, error)")
	flags.StringVar(&outputFormat, "output", "text", "result format (text, json)")
	flags.StringVar(&color, "color", "auto", "color mode for results and progress (auto, always, never)")
	flags.BoolVar(&quiet, "quiet", false, "suppress non-essential output")
	flags.DurationVar(&timeoutOverride, "timeout", 0, "override the timeout for the selected operation")
	runtime := &CommandRuntime{}
	var groupErr error
	var logResult logging.Result
	var cancelOperation context.CancelFunc
	var stopProgress func()
	var cleanupDependencies []func(context.Context) error
	var executionStarted bool
	cleanupTimeout := config.Defaults().Timeouts.Cleanup
	var cleanupComplete bool
	cleanup := func(parentContext context.Context) error {
		if cleanupComplete {
			return nil
		}
		cleanupComplete = true
		if stopProgress != nil {
			stopProgress()
			stopProgress = nil
		}
		if cancelOperation != nil {
			cancelOperation()
			cancelOperation = nil
		}
		var dependencyErr error
		if len(cleanupDependencies) > 0 {
			cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(parentContext), cleanupTimeout)
			for i := len(cleanupDependencies) - 1; i >= 0; i-- {
				dependencyErr = errors.Join(dependencyErr, cleanupDependencies[i](cleanupContext))
			}
			cancel()
			cleanupDependencies = nil
			if dependencyErr != nil {
				dependencyErr = fmt.Errorf("cleanup application dependencies: %w", dependencyErr)
			}
		}
		var loggerErr error
		if logResult.Close != nil {
			close := logResult.Close
			logResult.Close = nil
			if err := close(); err != nil {
				loggerErr = fmt.Errorf("close logger: %w", err)
			}
		}
		return errors.Join(dependencyErr, loggerErr)
	}
	preRun := func(cmd *cobra.Command, _ []string) error {
		if groupErr != nil {
			return groupErr
		}
		if executionStarted {
			return errors.New("command tree already executed; create a new root command")
		}
		executionStarted = true
		if cmd.HasSubCommands() || isHelpOrCompletion(cmd) {
			return nil
		}
		workingDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("find working directory: %w", err)
		}
		files, err := config.DiscoverFiles(workingDir, configID, configPath)
		if err != nil {
			return app.Classify(app.ErrValidation, err)
		}
		flagValues := map[string]config.FlagValue{
			"log-level": {Value: logLevel, Changed: flags.Lookup("log-level").Changed, Source: "--log-level"},
			"output":    {Value: outputFormat, Changed: flags.Lookup("output").Changed, Source: "--output"},
			"color":     {Value: color, Changed: flags.Lookup("color").Changed, Source: "--color"},
			"quiet":     {Value: quiet, Changed: flags.Lookup("quiet").Changed, Source: "--quiet"},
		}
		if registration.FlagValues != nil {
			for key, value := range registration.FlagValues(cmd) {
				flagValues[key] = value
			}
		}
		loaded, err := config.LoadWithSettings(files, flagValues, registration.Settings)
		if err != nil {
			return app.Classify(app.ErrValidation, err)
		}
		if registration.PrepareApplication != nil {
			application, err := registration.PrepareApplication(loaded.Application)
			if err != nil {
				return fmt.Errorf("prepare application settings: %w", err)
			}
			loaded.Application = application
		}
		runtime.Configuration = loaded
		policy := runtime.selectedPolicy(cmd)
		cleanupTimeout = loaded.Config.Timeouts.Cleanup
		noColor := os.Getenv("NO_COLOR") != ""
		runtime.Renderer = output.Renderer{
			Writer: cmd.OutOrStdout(),
			Options: output.Options{
				Format:    output.Format(loaded.Config.Output),
				Color:     output.ColorMode(loaded.Config.Color),
				Terminals: output.DetectTerminalCapabilities(cmd.OutOrStdout(), cmd.ErrOrStderr()),
				NoColor:   noColor,
			},
		}
		var configuredTimeout *time.Duration
		if flags.Lookup("timeout").Changed {
			if timeoutOverride <= 0 {
				return app.Classify(app.ErrValidation, fmt.Errorf("invalid --timeout %s: must be positive", timeoutOverride))
			}
			configuredTimeout = &timeoutOverride
		} else if policy.Timeout != nil {
			configuredTimeout = policy.Timeout(cmd)
		} else if registration.OperationTimeout != nil {
			configuredTimeout = registration.OperationTimeout(cmd, loaded.Application)
		}
		applyTimeout := func(command *cobra.Command, configured *time.Duration) error {
			selected := configured
			if selected == nil {
				return nil
			}
			if *selected <= 0 {
				return app.Classify(app.ErrValidation, fmt.Errorf("invalid operation timeout %s: must be positive", *selected))
			}
			parentContext := command.Context()
			operationContext, cancel := context.WithTimeout(parentContext, *selected)
			command.SetContext(operationContext)
			cancelOperation = func() { cancel(); command.SetContext(parentContext) }
			return nil
		}
		if err := applyTimeout(cmd, configuredTimeout); err != nil {
			return err
		}
		if cmd.Annotations[inspectionOnly] != "" {
			return nil
		}
		logResult, err = logging.New(loaded.Config.Log, cmd.ErrOrStderr())
		if err != nil {
			return fmt.Errorf("initialize logging: %w", err)
		}
		var progressReporter app.ProgressReporter = output.NewPlainProgressReporter(cmd.ErrOrStderr())
		switch {
		case loaded.Config.Quiet:
			progressReporter = output.NoopProgressReporter{}
		case runtime.Renderer.Options.ProgressColorEnabled():
			styled := output.NewStyledProgressReporter(cmd.ErrOrStderr(), runtime.Renderer.Options.Terminals.Stderr)
			stopProgress = styled.Close
			progressReporter = styled
		}
		runtime.Logger = logResult.Logger
		runtime.Progress = progressReporter
		if policy.Setup.Declared && policy.Setup.Run != nil {
			if err := policy.Setup.Run(cmd, runtime); err != nil {
				return fmt.Errorf("setup selected command: %w", err)
			}
		}
		return nil
	}
	root.PersistentPostRunE = func(cmd *cobra.Command, _ []string) error {
		return cleanup(cmd.Context())
	}
	runtime.registerCleanup = func(cleanup func(context.Context) error) {
		cleanupDependencies = append(cleanupDependencies, cleanup)
	}
	if registration.RegisterCommands != nil {
		registration.RegisterCommands(root, runtime)
	}
	groupErr = setApplicationGroupHelp(root)
	if root.PersistentPreRun != nil || root.PersistentPreRunE != nil {
		groupErr = errors.Join(groupErr, errors.New("root command must not define persistent pre-run hooks; use a setup policy"))
	}
	root.PersistentPreRun = nil
	root.PersistentPreRunE = preRun
	root.AddCommand(
		newConfigCommand(
			func() []config.Entry { return runtime.Configuration.Entries() },
			func() output.Renderer { return runtime.Renderer },
		),
	)
	// The root Args validator stops Cobra's Find from rejecting an unknown help topic itself.
	root.InitDefaultHelpCmd()
	help, _, _ := root.Find([]string{"help"})
	help.Args = func(cmd *cobra.Command, args []string) error {
		if target, rest, _ := cmd.Root().Find(args); len(rest) > 0 {
			return app.Classify(app.ErrValidation, unknownCommandError(target, rest[0]))
		}
		return nil
	}
	return root
}

func setApplicationGroupHelp(root *cobra.Command) error {
	var groupErr error
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if command.PersistentPreRun != nil || command.PersistentPreRunE != nil {
			groupErr = errors.Join(groupErr, fmt.Errorf(
				"command %q must not define persistent pre-run hooks; use a setup policy", command.CommandPath(),
			))
			command.PersistentPreRun = nil
			command.PersistentPreRunE = nil
		}
		children := command.Commands()
		if len(children) > 0 {
			if command.Run != nil || command.RunE != nil {
				handler := "Run"
				if command.RunE != nil {
					handler = "RunE"
				}
				groupErr = errors.Join(groupErr, fmt.Errorf("command group %q must not define %s; groups only show help", command.CommandPath(), handler))
			} else {
				rejectUnknownSubcommands(command)
			}
		}
		for _, child := range children {
			visit(child)
		}
	}
	for _, command := range root.Commands() {
		for _, name := range append([]string{command.Name()}, command.Aliases...) {
			switch name {
			case "help", "completion", "config", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
				groupErr = errors.Join(groupErr, fmt.Errorf("root command name %q is reserved", name))
			}
		}
		visit(command)
	}
	return groupErr
}

// rejectUnknownSubcommands makes a command that only groups subcommands runnable. Cobra skips Args
// on a non-runnable command and prints its help with exit 0, so a mistyped subcommand would succeed.
func rejectUnknownSubcommands(cmd *cobra.Command) {
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return app.Classify(app.ErrValidation, unknownCommandError(cmd, args[0]))
		}
		return nil
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return cmd.Help() }
}

// unknownCommandError builds the unknown-command error by hand: setting Args turns off Cobra's own check,
// whose error would be unclassified (exit 1) and whose suggestion helper is unexported.
func unknownCommandError(cmd *cobra.Command, name string) error {
	message := fmt.Sprintf("unknown command %q for %q", name, cmd.CommandPath())
	if !cmd.DisableSuggestions {
		if cmd.SuggestionsMinimumDistance <= 0 {
			cmd.SuggestionsMinimumDistance = 2
		}
		if suggestions := cmd.SuggestionsFor(name); len(suggestions) > 0 {
			message += "\n\nDid you mean this?\n\t" + strings.Join(suggestions, "\n\t") + "\n"
		}
	}
	return errors.New(message)
}

// validationArgs classifies a positional-argument failure as a validation error (exit 2).
func validationArgs(inner cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		return app.Classify(app.ErrValidation, inner(cmd, args))
	}
}

func isHelpOrCompletion(cmd *cobra.Command) bool {
	root := cmd.Root()
	for cmd.Parent() != nil && cmd.Parent() != root {
		cmd = cmd.Parent()
	}
	if cmd.Parent() != root {
		return false
	}
	switch cmd.Name() {
	case "help", "completion", cobra.ShellCompRequestCmd:
		return true
	default:
		return false
	}
}
