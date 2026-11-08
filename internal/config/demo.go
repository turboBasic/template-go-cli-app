package config

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// DemoConfig holds settings for the reference demo commands.
type DemoConfig struct {
	SecretCredential SecretCredentialConfig `koanf:"secret_credential"`
}

// SecretCredentialConfig holds settings for the secret-credential demo.
type SecretCredentialConfig struct {
	// Reference is a secret reference such as op://vault/item/field, never the secret itself.
	Reference string `koanf:"reference"`
}

// RetryConfig holds retry policies used by the reference demos.
type RetryConfig struct {
	Demo DemoRetryConfig `koanf:"demo"`
}

// DemoRetryConfig holds retry policies for the reference demo commands.
type DemoRetryConfig struct {
	FlakyDependency RetryPolicyConfig `koanf:"flaky_dependency"`
}

// RetryPolicyConfig is a bounded retry policy for one configured operation.
type RetryPolicyConfig struct {
	MaxAttempts  int           `koanf:"max_attempts"`
	InitialDelay time.Duration `koanf:"initial_delay"`
	MaxDelay     time.Duration `koanf:"max_delay"`
	Jitter       float64       `koanf:"jitter"`
}

// DemoTimeouts configures timeouts for the reference demo commands.
type DemoTimeouts struct {
	StructuredOutput *time.Duration `koanf:"structured_output"`
	NotFound         *time.Duration `koanf:"not_found"`
	SecretCredential *time.Duration `koanf:"secret_credential"`
}

func demoRetryDefaults() RetryConfig {
	// The demo retries by default so it succeeds out of the box; a run takes about 9s with the spinner visible.
	return RetryConfig{Demo: DemoRetryConfig{FlakyDependency: RetryPolicyConfig{MaxAttempts: 3, InitialDelay: 3 * time.Second, MaxDelay: 10 * time.Second}}}
}

func (c Config) validateDemo() error {
	for _, setting := range []struct {
		key     string
		timeout *time.Duration
	}{
		{key: "timeouts.demo.structured_output", timeout: c.Timeouts.Demo.StructuredOutput},
		{key: "timeouts.demo.not_found", timeout: c.Timeouts.Demo.NotFound},
		{key: "timeouts.demo.secret_credential", timeout: c.Timeouts.Demo.SecretCredential},
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
