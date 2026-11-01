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

// ResourceFactory builds the command's application service and its dependency cleanup.
type ResourceFactory func(context.Context, config.Config, *slog.Logger, app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error)

// BuildInfo is the version metadata embedded in the executable.
type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

// NewRootCommand builds an independently configurable command tree.
func NewRootCommand(reader app.ResourceReader, build BuildInfo, names []string) *cobra.Command {
	return NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger, progressReporter app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(reader, logger, progressReporter), nil, nil
	}, build, names)
}

// NewRootCommandWithFactory builds a command tree that loads config before creating application dependencies.
func NewRootCommandWithFactory(factory ResourceFactory, build BuildInfo, names []string) *cobra.Command {
	names = append([]string(nil), names...)
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
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return app.Classify(app.ErrValidation, err)
	})
	root.SetVersionTemplate("{{printf \"%s version %s (commit %s, built %s)\\n\" .Name .Version (index .Annotations \"commit\") (index .Annotations \"buildDate\")}}")
	var configPath, logLevel, outputFormat, color string
	var timeoutOverride time.Duration
	var quiet bool
	flags := root.PersistentFlags()
	flags.StringVar(&configPath, "config", "", "configuration file path")
	flags.StringVar(&logLevel, "log-level", "warn", "logging level (debug, info, warn, error)")
	flags.StringVar(&outputFormat, "output", "text", "result format (text, json)")
	flags.StringVar(&color, "color", "auto", "color mode (auto, always, never)")
	flags.BoolVar(&quiet, "quiet", false, "suppress non-essential output")
	flags.DurationVar(&timeoutOverride, "timeout", 0, "override the timeout for the selected resource operation")
	var resources *app.ResourceService
	var renderer output.Renderer
	var logResult logging.Result
	var cancelOperation context.CancelFunc
	var cleanupDependencies func(context.Context) error
	cleanupTimeout := config.Defaults().Timeouts.Cleanup
	var cleanupComplete bool
	cleanup := func(parentContext context.Context) error {
		if cleanupComplete {
			return nil
		}
		cleanupComplete = true
		if cancelOperation != nil {
			cancelOperation()
			cancelOperation = nil
		}
		var dependencyErr error
		if cleanupDependencies != nil {
			cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(parentContext), cleanupTimeout)
			dependencyErr = cleanupDependencies(cleanupContext)
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
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if err := cleanup(cmd.Context()); err != nil {
			return err
		}
		cleanupComplete = false
		resources = nil
		renderer = output.Renderer{}
		if isHelpOrCompletion(cmd) {
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
		if cmd.Parent() != nil && cmd.Parent().Name() == "resource" && flags.Lookup("timeout").Changed {
			key := "timeouts.resource." + cmd.Name()
			flagValues[key] = config.FlagValue{Value: timeoutOverride, Changed: true, Source: "--timeout"}
		}
		loaded, err := config.Load(files, flagValues)
		if err != nil {
			return app.Classify(app.ErrValidation, err)
		}
		cleanupTimeout = loaded.Config.Timeouts.Cleanup
		_, noColor := os.LookupEnv("NO_COLOR")
		renderer = output.Renderer{
			Writer: cmd.OutOrStdout(),
			Options: output.Options{
				Format:    output.Format(loaded.Config.Output),
				Color:     output.ColorMode(loaded.Config.Color),
				Terminals: output.DetectTerminalCapabilities(cmd.OutOrStdout(), cmd.ErrOrStderr()),
				NoColor:   noColor,
			},
		}
		logResult, err = logging.New(loaded.Config.Log)
		if err != nil {
			return fmt.Errorf("initialize logging: %w", err)
		}
		var progressReporter app.ProgressReporter = output.NewPlainProgressReporter(cmd.ErrOrStderr())
		if loaded.Config.Quiet {
			progressReporter = output.NoopProgressReporter{}
		}
		if timeout := resourceOperationTimeout(cmd, loaded.Config.Timeouts.Resource); timeout != nil {
			parentContext := cmd.Context()
			operationContext, cancel := context.WithTimeout(parentContext, *timeout)
			cmd.SetContext(operationContext)
			cancelOperation = func() {
				cancel()
				cmd.SetContext(parentContext)
			}
		}
		if factory == nil {
			return errors.Join(fmt.Errorf("resource factory is required"), cleanup(cmd.Context()))
		}
		resources, cleanupDependencies, err = factory(cmd.Context(), loaded.Config, logResult.Logger, progressReporter)
		if err != nil {
			return errors.Join(fmt.Errorf("create application dependencies: %w", err), cleanup(cmd.Context()))
		}
		if resources == nil {
			return errors.Join(fmt.Errorf("resource factory returned no service"), cleanup(cmd.Context()))
		}
		return nil
	}
	root.PersistentPostRunE = func(cmd *cobra.Command, _ []string) error {
		return cleanup(cmd.Context())
	}
	resource := &cobra.Command{Use: "resource", Short: "Manage resources"}
	getResources := func() *app.ResourceService { return resources }
	getRenderer := func() output.Renderer { return renderer }
	list := newResourceListCommand(getResources, getRenderer)
	status := newResourceStatusCommand(getResources, getRenderer, names)
	apply := newResourceApplyCommand(getResources, getRenderer, names)
	closeAfterRun(list, cleanup)
	closeAfterRun(status, cleanup)
	closeAfterRun(apply, cleanup)
	resource.AddCommand(list, status, apply)
	root.AddCommand(resource)
	return root
}

func resourceOperationTimeout(cmd *cobra.Command, timeouts config.ResourceTimeouts) *time.Duration {
	if cmd.Parent() == nil || cmd.Parent().Name() != "resource" {
		return nil
	}
	switch cmd.Name() {
	case "list":
		return timeouts.List
	case "status":
		return timeouts.Status
	case "apply":
		return timeouts.Apply
	default:
		return nil
	}
}

func closeAfterRun(command *cobra.Command, cleanup func(context.Context) error) {
	run := command.RunE
	command.RunE = func(cmd *cobra.Command, args []string) error {
		return errors.Join(run(cmd, args), cleanup(cmd.Context()))
	}
}

func isHelpOrCompletion(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		if current.Name() == "help" || current.Name() == "completion" || strings.HasPrefix(current.Name(), "__complete") {
			return true
		}
	}
	return false
}

func newResourceListCommand(getResources func() *app.ResourceService, getRenderer func() output.Renderer) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List resources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			items, err := getResources().List(cmd.Context())
			if err != nil {
				return err
			}
			return getRenderer().RenderList(items)
		},
	}
}

func newResourceStatusCommand(getResources func() *app.ResourceService, getRenderer func() output.Renderer, names []string) *cobra.Command {
	return &cobra.Command{
		Use:               "status <name>",
		Short:             "Show a resource status",
		Args:              validateResourceName(names),
		ValidArgsFunction: completeResourceNames(names),
		RunE: func(cmd *cobra.Command, args []string) error {
			resource, err := getResources().Status(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return getRenderer().RenderResource(resource)
		},
	}
}

func newResourceApplyCommand(getResources func() *app.ResourceService, getRenderer func() output.Renderer, names []string) *cobra.Command {
	return &cobra.Command{
		Use:               "apply <name>",
		Short:             "Apply a resource",
		Args:              validateResourceName(names),
		ValidArgsFunction: completeResourceNames(names),
		RunE: func(cmd *cobra.Command, args []string) error {
			resource, err := getResources().Apply(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return getRenderer().RenderResource(resource)
		},
	}
}

func completeResourceNames(names []string) cobra.CompletionFunc {
	return func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var matches []string
		for _, name := range names {
			if strings.HasPrefix(name, toComplete) {
				matches = append(matches, name)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

func validateResourceName(names []string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			return app.Classify(app.ErrValidation, err)
		}
		for _, name := range names {
			if args[0] == name {
				return nil
			}
		}
		return app.Classify(app.ErrValidation, fmt.Errorf("unknown resource %q; valid names are %s", args[0], strings.Join(names, ", ")))
	}
}
