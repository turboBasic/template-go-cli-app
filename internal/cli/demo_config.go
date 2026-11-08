package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
)

const ownTimeout = "own-timeout"

func demoFlagValues(cmd *cobra.Command, timeoutChanged bool, timeoutOverride time.Duration) map[string]config.FlagValue {
	values := make(map[string]config.FlagValue)
	if name := demoTimeoutName(cmd); name != "" && cmd.Annotations[ownTimeout] == "" && timeoutChanged {
		values["timeouts.demo."+name] = config.FlagValue{Value: timeoutOverride, Changed: true, Source: "--timeout"}
	}
	if flag := cmd.Flags().Lookup("max-attempts"); flag != nil && flag.Changed {
		attempts, _ := cmd.Flags().GetInt("max-attempts")
		values["retry.demo.flaky_dependency.max_attempts"] = config.FlagValue{Value: attempts, Changed: true, Source: "--max-attempts"}
	}
	return values
}

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

func flagTimeoutContext(cmd *cobra.Command) (context.Context, context.CancelFunc, error) {
	ctx := cmd.Context()
	if !cmd.Flags().Changed("timeout") {
		return ctx, func() {}, nil
	}
	timeout, err := cmd.Flags().GetDuration("timeout")
	if err != nil || timeout <= 0 {
		return nil, nil, app.Classify(app.ErrValidation, fmt.Errorf("invalid --timeout %s: must be positive", timeout))
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	return ctx, cancel, nil
}
