package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
)

// BuildInfo is the version metadata embedded in the executable.
type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

// NewRootCommand builds an independently configurable command tree.
func NewRootCommand(resources *app.ResourceService, build BuildInfo) *cobra.Command {
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
	var config, logLevel, output, color string
	var quiet bool
	flags := root.PersistentFlags()
	flags.StringVar(&config, "config", "", "configuration file path")
	flags.StringVar(&logLevel, "log-level", "warn", "logging level (debug, info, warn, error)")
	flags.StringVar(&output, "output", "text", "result format (text, json)")
	flags.StringVar(&color, "color", "auto", "color mode (auto, always, never)")
	flags.BoolVar(&quiet, "quiet", false, "suppress non-essential output")
	root.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		if !oneOf(logLevel, "debug", "info", "warn", "error") {
			return fmt.Errorf("invalid --log-level %q: must be debug, info, warn, or error", logLevel)
		}
		if !oneOf(output, "text", "json") {
			return fmt.Errorf("invalid --output %q: must be text or json", output)
		}
		if !oneOf(color, "auto", "always", "never") {
			return fmt.Errorf("invalid --color %q: must be auto, always, or never", color)
		}
		if output == "json" && color == "always" {
			return fmt.Errorf("--color always cannot be used with --output json")
		}
		return nil
	}
	resource := &cobra.Command{Use: "resource", Short: "Manage resources"}
	resource.AddCommand(newResourceListCommand(resources), newResourceStatusCommand(resources), newResourceApplyCommand(resources))
	root.AddCommand(resource)
	return root
}

func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

func newResourceListCommand(resources *app.ResourceService) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List resources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			items, err := resources.List(cmd.Context())
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

func newResourceStatusCommand(resources *app.ResourceService) *cobra.Command {
	return &cobra.Command{
		Use:               "status <name>",
		Short:             "Show a resource status",
		Args:              validateResourceName,
		ValidArgsFunction: completeResourceNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			resource, err := resources.Status(cmd.Context(), args[0])
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

func newResourceApplyCommand(resources *app.ResourceService) *cobra.Command {
	return &cobra.Command{
		Use:               "apply <name>",
		Short:             "Apply a resource",
		Args:              validateResourceName,
		ValidArgsFunction: completeResourceNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			resource, err := resources.Apply(cmd.Context(), args[0])
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

func completeResourceNames(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var matches []string
	for _, name := range resourceNames() {
		if strings.HasPrefix(name, toComplete) {
			matches = append(matches, name)
		}
	}
	return matches, cobra.ShellCompDirectiveNoFileComp
}

func validateResourceName(cmd *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(cmd, args); err != nil {
		return err
	}
	for _, name := range resourceNames() {
		if args[0] == name {
			return nil
		}
	}
	return fmt.Errorf("unknown resource %q; valid names are %s", args[0], strings.Join(resourceNames(), ", "))
}

func resourceNames() []string {
	return []string{"example-1", "example-2"}
}
