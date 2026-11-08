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
	rejectUnknownSubcommands(root)
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
	flags.StringVar(&color, "color", "auto", "result color mode (auto, always, never)")
	flags.BoolVar(&quiet, "quiet", false, "suppress non-essential output")
	flags.DurationVar(&timeoutOverride, "timeout", 0, "override the timeout for the selected operation")
	var resources *app.ResourceService
	var renderer output.Renderer
	var loadResult config.LoadResult
	var logResult logging.Result
	var cancelOperation context.CancelFunc
	var stopProgress func()
	var cleanupDependencies func(context.Context) error
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
		stopProgress = nil
		renderer = output.Renderer{}
		loadResult = config.LoadResult{}
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
		if name := demoTimeoutName(cmd); name != "" && flags.Lookup("timeout").Changed {
			// A demo without a schema key, long-running, is skipped here and applies --timeout itself.
			flagValues["timeouts.demo."+name] = config.FlagValue{Value: timeoutOverride, Changed: true, Source: "--timeout"}
		}
		loaded, err := config.Load(files, flagValues)
		if err != nil {
			return app.Classify(app.ErrValidation, err)
		}
		loadResult = loaded
		cleanupTimeout = loaded.Config.Timeouts.Cleanup
		noColor := os.Getenv("NO_COLOR") != ""
		renderer = output.Renderer{
			Writer: cmd.OutOrStdout(),
			Options: output.Options{
				Format:    output.Format(loaded.Config.Output),
				Color:     output.ColorMode(loaded.Config.Color),
				Terminals: output.DetectTerminalCapabilities(cmd.OutOrStdout(), cmd.ErrOrStderr()),
				NoColor:   noColor,
			},
		}
		if cmd.Annotations[inspectionOnly] != "" {
			return nil
		}
		logResult, err = logging.New(loaded.Config.Log)
		if err != nil {
			return fmt.Errorf("initialize logging: %w", err)
		}
		var progressReporter app.ProgressReporter = output.NewPlainProgressReporter(cmd.ErrOrStderr())
		switch {
		case loaded.Config.Quiet:
			progressReporter = output.NoopProgressReporter{}
		case renderer.Options.ProgressColorEnabled():
			styled := output.NewStyledProgressReporter(cmd.ErrOrStderr(), renderer.Options.Terminals.Stderr)
			stopProgress = styled.Close
			progressReporter = styled
		}
		if timeout := demoOperationTimeout(cmd, loaded.Config.Timeouts.Demo); timeout != nil {
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
	getResources := func() *app.ResourceService { return resources }
	getRenderer := func() output.Renderer { return renderer }
	demo := newDemoCommand(getResources, getRenderer, names)
	for _, command := range demo.Commands() {
		closeAfterRun(command, cleanup)
	}
	root.AddCommand(demo, newConfigCommand(func() []config.Entry { return loadResult.Entries() }, getRenderer, cleanup))
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

// demoTimeoutName is the last element of a demo command's timeouts.demo.<name> key, or "" outside the group.
func demoTimeoutName(cmd *cobra.Command) string {
	if cmd.Parent() == nil || cmd.Parent().Name() != "demo" {
		return ""
	}
	return strings.ReplaceAll(cmd.Name(), "-", "_")
}

func demoOperationTimeout(cmd *cobra.Command, timeouts config.DemoTimeouts) *time.Duration {
	switch demoTimeoutName(cmd) {
	case "structured_output":
		return timeouts.StructuredOutput
	case "not_found":
		return timeouts.NotFound
	case "secret_credential":
		return timeouts.SecretCredential
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

// unknownCommandError reproduces Cobra's own unknown-command message, suggestions included.
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
	for current := cmd; current != nil; current = current.Parent() {
		if current.Name() == "help" || current.Name() == "completion" || strings.HasPrefix(current.Name(), "__complete") {
			return true
		}
	}
	return false
}

func newDemoStructuredOutputCommand(getResources func() *app.ResourceService, getRenderer func() output.Renderer) *cobra.Command {
	return &cobra.Command{
		Use:   "structured-output",
		Short: "Demonstrate text and JSON results written to stdout",
		Long: "Demonstrates the result contract: the resource list is a human table by default and a JSON array with\n" +
			"--output json, always on stdout. Look at the split: redirect stdout and only the result arrives, with no\n" +
			"progress or diagnostics mixed in.",
		Args: validationArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			items, err := getResources().List(cmd.Context())
			if err != nil {
				return err
			}
			return getRenderer().RenderList(items)
		},
	}
}

func newDemoNotFoundCommand(getResources func() *app.ResourceService, getRenderer func() output.Renderer, names []string) *cobra.Command {
	return &cobra.Command{
		Use:   "not-found <name>",
		Short: "Demonstrate an actionable not-found error with shell completion of names",
		Long: "Demonstrates the error contract: an unknown name fails with exit 1 and a message that names it and lists the\n" +
			"known names. Look at the message, then press Tab after the command: the shell completes the known names.",
		Args:              validationArgs(cobra.ExactArgs(1)),
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

func newDemoSecretCredentialCommand(getResources func() *app.ResourceService, getRenderer func() output.Renderer, names []string) *cobra.Command {
	return &cobra.Command{
		Use:   "secret-credential <name>",
		Short: "Demonstrate a secret reference resolved at run time and never printed",
		Long: "Demonstrates the secret contract: set demo.secret_credential.reference (or\n" +
			"CLI_APP_DEMO_SECRET_CREDENTIAL_REFERENCE) to a scheme://reference such as an op:// item. The reference is\n" +
			"resolved only when this command runs, and the secret value never reaches output, logs or `config show`.",
		Args:              validationArgs(cobra.ExactArgs(1)),
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

// newDemoCommand groups the template's demonstrations of its contracts. An application built from the
// template deletes the group, and every demo with it.
func newDemoCommand(getResources func() *app.ResourceService, getRenderer func() output.Renderer, names []string) *cobra.Command {
	demo := &cobra.Command{
		Use:   "demo",
		Short: "Demonstrations of the template's contracts; delete this group in your application",
		Long: "Demonstrations of the template's contracts, kept so a developer can see each one working and copy it.\n" +
			"They are not part of an application built from the template: delete this group, and the code it calls.",
	}
	rejectUnknownSubcommands(demo)
	demo.AddCommand(newDemoStructuredOutputCommand(getResources, getRenderer), newDemoNotFoundCommand(getResources, getRenderer, names), newDemoSecretCredentialCommand(getResources, getRenderer, names))
	demo.AddCommand(&cobra.Command{
		Use:   "long-running",
		Short: "Demonstrate the determinate progress bar with a counted operation of about 30 seconds",
		Long: "Demonstrates determinate progress: a counted operation reports each step with its current and total, and a\n" +
			"color-capable terminal shows a bar that fills. Redirected stderr, --output json and --quiet keep plain lines or\n" +
			"silence. The length is fixed in the template; --timeout bounds it and Ctrl-C cancels it.",
		Args: validationArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if cmd.Flags().Changed("timeout") {
				timeout, err := cmd.Flags().GetDuration("timeout")
				if err != nil || timeout <= 0 {
					return app.Classify(app.ErrValidation, fmt.Errorf("invalid --timeout %s: must be positive", timeout))
				}
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
			return getResources().LongRunning(ctx)
		},
	})
	return demo
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
