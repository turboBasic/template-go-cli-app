package cli

import (
	"fmt"
	"math"
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

// DemoConfig holds configuration used only by the reference demo commands.
type DemoConfig struct {
	Demo     demoSettings        `koanf:"demo"`
	Timeouts demoTimeoutSettings `koanf:"timeouts"`
	Retry    demoRetrySettings   `koanf:"retry"`
}

type demoSettings struct {
	SecretCredential demoSecretCredential `koanf:"secret_credential"`
}
type demoSecretCredential struct {
	Reference string `koanf:"reference"`
}
type demoTimeoutSettings struct {
	Demo demoTimeouts `koanf:"demo"`
}
type demoTimeouts struct {
	StructuredOutput *time.Duration `koanf:"structured_output"`
	NotFound         *time.Duration `koanf:"not_found"`
	SecretCredential *time.Duration `koanf:"secret_credential"`
}
type demoRetrySettings struct {
	Demo demoRetryConfig `koanf:"demo"`
}
type demoRetryConfig struct {
	FlakyDependency demoRetryPolicy `koanf:"flaky_dependency"`
}
type demoRetryPolicy struct {
	MaxAttempts  int           `koanf:"max_attempts"`
	InitialDelay time.Duration `koanf:"initial_delay"`
	MaxDelay     time.Duration `koanf:"max_delay"`
	Jitter       float64       `koanf:"jitter"`
}

func demoDefaults() DemoConfig {
	return DemoConfig{Retry: demoRetrySettings{Demo: demoRetryConfig{FlakyDependency: demoRetryPolicy{MaxAttempts: 3, InitialDelay: 3 * time.Second, MaxDelay: 10 * time.Second}}}}
}

// DemoApplicationSettings registers the reference demo's defaults and validation.
func DemoApplicationSettings() config.ApplicationSettings {
	return config.ApplicationSettings{Defaults: demoDefaults(), Validate: func(value any) error { return validateDemoConfig(value.(DemoConfig)) }}
}

func validateDemoConfig(c DemoConfig) error {
	for _, setting := range []struct {
		key     string
		timeout *time.Duration
	}{
		{"timeouts.demo.structured_output", c.Timeouts.Demo.StructuredOutput},
		{"timeouts.demo.not_found", c.Timeouts.Demo.NotFound},
		{"timeouts.demo.secret_credential", c.Timeouts.Demo.SecretCredential},
	} {
		if setting.timeout != nil && *setting.timeout <= 0 {
			return fmt.Errorf("invalid %s %s: must be positive", setting.key, *setting.timeout)
		}
	}
	policy := c.Retry.Demo.FlakyDependency
	if policy.MaxAttempts < 1 {
		return fmt.Errorf("invalid retry.demo.flaky_dependency.max_attempts: must be positive")
	}
	if policy.InitialDelay < 0 || policy.MaxDelay < 0 || (policy.MaxAttempts > 1 && (policy.InitialDelay == 0 || policy.MaxDelay < policy.InitialDelay)) {
		return fmt.Errorf("invalid retry.demo.flaky_dependency delays: require 0 < initial <= max for multiple attempts")
	}
	if math.IsNaN(policy.Jitter) || policy.Jitter < 0 || policy.Jitter > 1 {
		return fmt.Errorf("invalid retry.demo.flaky_dependency.jitter: must be between 0 and 1")
	}
	if ref := c.Demo.SecretCredential.Reference; ref != "" {
		if scheme, rest, ok := strings.Cut(ref, "://"); !ok || scheme == "" || rest == "" {
			return fmt.Errorf("invalid demo.secret_credential.reference: must be a scheme://reference, not the secret itself")
		}
	}
	return nil
}

func configuredOperationTimeout(cmd *cobra.Command, timeouts demoTimeouts) *time.Duration {
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
