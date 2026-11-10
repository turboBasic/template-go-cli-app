package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/output"
)

func newDemoStructuredOutputCommand(group *demoResourceGroup) *cobra.Command {
	return &cobra.Command{
		Use:   "structured-output",
		Short: "Demonstrate text and JSON results written to stdout",
		Long: "Demonstrates the result contract: the resource list is a human table by default and a JSON array with\n" +
			"--output json, always on stdout. Look at the split: redirect stdout and only the result arrives, with no\n" +
			"progress or diagnostics mixed in.",
		Args: validationArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			resources, err := group.get(cmd.Context())
			if err != nil {
				return err
			}
			items, err := resources.List(cmd.Context())
			if err != nil {
				return err
			}
			return group.runtime.Renderer.RenderResult(
				demoResourceResultsFor(items),
				func(w io.Writer, options output.Options) error {
					return renderDemoResources(w, options, items)
				},
			)
		},
	}
}

func newDemoNotFoundCommand(group *demoResourceGroup, names []string) *cobra.Command {
	return &cobra.Command{
		Use:   "not-found <name>",
		Short: "Demonstrate an actionable not-found error with shell completion of names",
		Long: "Demonstrates the error contract: an unknown name fails with exit 1 and a message that names it and lists the\n" +
			"known names. Look at the message, then press Tab after the command: the shell completes the known names.",
		Args:              validationArgs(cobra.ExactArgs(1)),
		ValidArgsFunction: completeResourceNames(names),
		RunE: func(cmd *cobra.Command, args []string) error {
			resources, err := group.get(cmd.Context())
			if err != nil {
				return err
			}
			resource, err := resources.Status(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return group.runtime.Renderer.RenderResult(
				demoResourceResultFor(resource),
				func(w io.Writer, options output.Options) error {
					return renderDemoResource(w, options, resource)
				},
			)
		},
	}
}

func newDemoSecretCredentialCommand(group *demoResourceGroup, names []string) *cobra.Command {
	return &cobra.Command{
		Use:   "secret-credential <name>",
		Short: "Demonstrate a secret reference resolved at run time and never printed",
		Long: "Demonstrates the secret contract: set demo.secret_credential.reference (or\n" +
			"CLI_APP_DEMO_SECRET_CREDENTIAL_REFERENCE) to a scheme://reference such as an op:// item. The reference is\n" +
			"resolved only when this command runs, and the secret value never reaches output, logs or `config show`.",
		Args:              validationArgs(cobra.ExactArgs(1)),
		ValidArgsFunction: completeResourceNames(names),
		RunE: func(cmd *cobra.Command, args []string) error {
			resources, err := group.get(cmd.Context())
			if err != nil {
				return err
			}
			resource, err := resources.Apply(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return group.runtime.Renderer.RenderResult(
				demoResourceResultFor(resource),
				func(w io.Writer, options output.Options) error {
					return renderDemoResource(w, options, resource)
				},
			)
		},
	}
}

func newDemoCommand(group *demoResourceGroup, names []string) *cobra.Command {
	demo := &cobra.Command{
		Use:   "demo",
		Short: "Demonstrations of the template's contracts; delete this group in your application",
		Long: "Demonstrations of the template's contracts, kept so a developer can see each one working and copy it.\n" +
			"They are not part of an application built from the template: delete this group, and the code it calls.",
	}
	rejectUnknownSubcommands(demo)
	demo.AddCommand(
		newDemoStructuredOutputCommand(group),
		newDemoNotFoundCommand(group, names),
		newDemoSecretCredentialCommand(group, names),
		newDemoFlakyDependencyCommand(group),
	)
	demo.AddCommand(&cobra.Command{
		Use:   "long-running",
		Short: "Demonstrate the determinate progress bar with a counted operation of about 30 seconds",
		Long: "Demonstrates determinate progress: a counted operation reports each step with its current and total, and a\n" +
			"color-capable terminal shows a bar that fills. Redirected stderr keeps plain lines and --quiet silences them. The\n" +
			"length is fixed in the template; --timeout bounds it and Ctrl-C cancels it.",
		Args: validationArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			resources, err := group.get(cmd.Context())
			if err != nil {
				return err
			}
			return resources.LongRunning(cmd.Context())
		},
	})
	return demo
}

func newDemoFlakyDependencyCommand(group *demoResourceGroup) *cobra.Command {
	defaultPolicy := demoDefaults().Retry.Demo.FlakyDependency
	command := &cobra.Command{
		Use:   "flaky-dependency",
		Short: "Demonstrate retries of a read-only dependency that fails transiently",
		Long: fmt.Sprintf(
			"Demonstrates the retry contract: the dependency this command reads fails %d times with an unavailable error, then\n"+
				"succeeds. The default policy allows %d attempts, waits %s before the first retry and doubles the wait up to %s,\n"+
				"so the spinner stays visible; each retry shows on stderr. Change the policy with\n"+
				"retry.demo.flaky_dependency.{max_attempts,initial_delay,max_delay,jitter} or the matching\n"+
				"CLI_APP_RETRY_DEMO_FLAKY_DEPENDENCY_* variables. --max-attempts overrides max_attempts for one run, and %d or\n"+
				"fewer shows the failure. --timeout bounds the retries.",
			app.FlakyFailures,
			defaultPolicy.MaxAttempts,
			defaultPolicy.InitialDelay,
			defaultPolicy.MaxDelay,
			app.FlakyFailures,
		),
		Args: validationArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			resources, err := group.get(cmd.Context())
			if err != nil {
				return err
			}
			resource, err := resources.FlakyDependency(cmd.Context())
			if err != nil {
				return err
			}
			return group.runtime.Renderer.RenderResult(
				demoResourceResultFor(resource),
				func(w io.Writer, options output.Options) error {
					return renderDemoResource(w, options, resource)
				},
			)
		},
	}
	command.Flags().Int("max-attempts", 0, "override retry.demo.flaky_dependency.max_attempts")
	return command
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
