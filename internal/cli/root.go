package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
)

// NewRootCommand builds the command tree around the resource use cases.
func NewRootCommand(resources *app.ResourceService, version string) *cobra.Command {
	root := &cobra.Command{
		Use:          "cli-app",
		Short:        "Manage simulated infrastructure resources",
		Version:      version,
		SilenceUsage: true,
	}
	resource := &cobra.Command{
		Use:   "resource",
		Short: "Manage resources",
	}
	list := &cobra.Command{
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
	resource.AddCommand(list)
	root.AddCommand(resource)
	return root
}
