package cli

import (
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/config"
)

func demoFlagValues(cmd *cobra.Command) map[string]config.FlagValue {
	values := make(map[string]config.FlagValue)
	if flag := cmd.Flags().Lookup("max-attempts"); flag != nil && flag.Changed {
		attempts, _ := cmd.Flags().GetInt("max-attempts")
		values["retry.demo.flaky_dependency.max_attempts"] = config.FlagValue{Value: attempts, Changed: true, Source: "--max-attempts"}
	}
	return values
}

func configuredOperationTimeout(cmd *cobra.Command, timeouts config.DemoTimeouts) *time.Duration {
	if cmd.Parent() == nil || cmd.Parent().Name() != "demo" {
		return nil
	}
	switch strings.ReplaceAll(cmd.Name(), "-", "_") {
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
