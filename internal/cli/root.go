package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
)

const configID = "cli-app"

// ResourceFactory builds the command's application service from effective configuration.
type ResourceFactory func(context.Context, config.Config) (*app.ResourceService, error)

// BuildInfo is the version metadata embedded in the executable.
type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

// NewRootCommand builds an independently configurable command tree.
func NewRootCommand(resources *app.ResourceService, build BuildInfo, names []string) *cobra.Command {
	return NewRootCommandWithFactory(func(context.Context, config.Config) (*app.ResourceService, error) {
		return resources, nil
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
	root.SetVersionTemplate("{{printf \"%s version %s (commit %s, built %s)\\n\" .Name .Version (index .Annotations \"commit\") (index .Annotations \"buildDate\")}}")
	var configPath, logLevel, output, color string
	var quiet bool
	flags := root.PersistentFlags()
	flags.StringVar(&configPath, "config", "", "configuration file path")
	flags.StringVar(&logLevel, "log-level", "warn", "logging level (debug, info, warn, error)")
	flags.StringVar(&output, "output", "text", "result format (text, json)")
	flags.StringVar(&color, "color", "auto", "color mode (auto, always, never)")
	flags.BoolVar(&quiet, "quiet", false, "suppress non-essential output")
	var resources *app.ResourceService
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		resources = nil
		if isHelpOrCompletion(cmd) {
			return nil
		}
		workingDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("find working directory: %w", err)
		}
		files, err := config.DiscoverFiles(workingDir, configID, configPath)
		if err != nil {
			return err
		}
		flagValues := map[string]config.FlagValue{
			"log-level": {Value: logLevel, Changed: flags.Lookup("log-level").Changed, Source: "--log-level"},
			"output":    {Value: output, Changed: flags.Lookup("output").Changed, Source: "--output"},
			"color":     {Value: color, Changed: flags.Lookup("color").Changed, Source: "--color"},
			"quiet":     {Value: quiet, Changed: flags.Lookup("quiet").Changed, Source: "--quiet"},
		}
		loaded, err := config.Load(files, flagValues)
		if err != nil {
			return err
		}
		if factory == nil {
			return fmt.Errorf("resource factory is required")
		}
		resources, err = factory(cmd.Context(), loaded.Config)
		if err != nil {
			return fmt.Errorf("create application dependencies: %w", err)
		}
		if resources == nil {
			return fmt.Errorf("resource factory returned no service")
		}
		return nil
	}
	resource := &cobra.Command{Use: "resource", Short: "Manage resources"}
	getResources := func() *app.ResourceService { return resources }
	resource.AddCommand(newResourceListCommand(getResources), newResourceStatusCommand(getResources, names), newResourceApplyCommand(getResources, names))
	root.AddCommand(resource)
	return root
}

func isHelpOrCompletion(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		if current.Name() == "help" || current.Name() == "completion" || strings.HasPrefix(current.Name(), "__complete") {
			return true
		}
	}
	return false
}

func newResourceListCommand(getResources func() *app.ResourceService) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List resources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			items, err := getResources().List(cmd.Context())
			if err != nil {
				return err
			}
			writer := tabwriter.NewWriter(cmd.OutOrStdout(), 12, 8, 0, ' ', 0)
			if _, err := fmt.Fprintln(writer, "NAME\tSTATUS"); err != nil {
				return fmt.Errorf("write resource list: %w", err)
			}
			for _, item := range items {
				if _, err := fmt.Fprintf(writer, "%s\t%s\n", item.Name, item.Status); err != nil {
					return fmt.Errorf("write resource list: %w", err)
				}
			}
			if err := writer.Flush(); err != nil {
				return fmt.Errorf("flush resource list: %w", err)
			}
			return nil
		},
	}
}

func newResourceStatusCommand(getResources func() *app.ResourceService, names []string) *cobra.Command {
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
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", resource.Name, resource.Status); err != nil {
				return fmt.Errorf("write resource status: %w", err)
			}
			return nil
		},
	}
}

func newResourceApplyCommand(getResources func() *app.ResourceService, names []string) *cobra.Command {
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
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", resource.Name, resource.Status); err != nil {
				return fmt.Errorf("write applied resource: %w", err)
			}
			return nil
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
			return err
		}
		for _, name := range names {
			if args[0] == name {
				return nil
			}
		}
		return fmt.Errorf("unknown resource %q; valid names are %s", args[0], strings.Join(names, ", "))
	}
}
