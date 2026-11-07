package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
	"github.com/turboBasic/template-go-cli-app/internal/logging"
	"github.com/turboBasic/template-go-cli-app/internal/output"
)

func newConfigCommand(getEntries func() []config.Entry, getRenderer func() output.Renderer, cleanup func(context.Context) error) *cobra.Command {
	group := &cobra.Command{Use: "config", Short: "Inspect configuration"}
	rejectUnknownSubcommands(group)
	show := &cobra.Command{
		Use:   "show [key]",
		Short: "Show the effective configuration and where each value comes from",
		Args:  validationArgs(cobra.MaximumNArgs(1)),
		RunE: func(_ *cobra.Command, args []string) error {
			entries := getEntries()
			if len(args) == 1 {
				entries = filterEntries(entries, args[0])
				if len(entries) == 0 {
					return app.Classify(app.ErrValidation, fmt.Errorf("no configuration key matches %q", args[0]))
				}
			}
			return getRenderer().RenderConfig(maskSensitive(entries))
		},
	}
	closeAfterRun(show, cleanup)
	group.AddCommand(show)
	return group
}

// filterEntries keeps the entry named key and every entry under it; a prefix must end on a segment boundary.
func filterEntries(entries []config.Entry, key string) []config.Entry {
	var kept []config.Entry
	for _, entry := range entries {
		if entry.Key == key || strings.HasPrefix(entry.Key, key+".") {
			kept = append(kept, entry)
		}
	}
	return kept
}

// maskSensitive returns a copy in which entries with a sensitive key carry the redaction marker.
func maskSensitive(entries []config.Entry) []config.Entry {
	masked := make([]config.Entry, len(entries))
	for i, entry := range entries {
		if logging.SensitiveKey(entry.Key) {
			entry.Value = logging.Redacted
		}
		masked[i] = entry
	}
	return masked
}
