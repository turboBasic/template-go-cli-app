package config

import (
	"fmt"
	"math"
	"strings"
	"time"
)

type testDemoConfig struct {
	Demo     testDemoSettings `koanf:"demo"`
	Timeouts testTimeoutRoot  `koanf:"timeouts"`
	Retry    testRetryRoot    `koanf:"retry"`
}
type testDemoSettings struct {
	SecretCredential testSecretCredential `koanf:"secret_credential"`
}
type testSecretCredential struct {
	Reference string `koanf:"reference"`
}
type testTimeoutRoot struct {
	Demo testDemoTimeouts `koanf:"demo"`
}
type testDemoTimeouts struct {
	StructuredOutput *time.Duration `koanf:"structured_output"`
	NotFound         *time.Duration `koanf:"not_found"`
	SecretCredential *time.Duration `koanf:"secret_credential"`
}
type testRetryRoot struct {
	Demo testDemoRetry `koanf:"demo"`
}
type testDemoRetry struct {
	FlakyDependency testRetryPolicy `koanf:"flaky_dependency"`
}
type testRetryPolicy struct {
	MaxAttempts  int           `koanf:"max_attempts"`
	InitialDelay time.Duration `koanf:"initial_delay"`
	MaxDelay     time.Duration `koanf:"max_delay"`
	Jitter       float64       `koanf:"jitter"`
}

func demoTestRegistration() ApplicationSettings {
	return ApplicationSettings{Defaults: testDemoConfig{Retry: testRetryRoot{Demo: testDemoRetry{FlakyDependency: testRetryPolicy{MaxAttempts: 3, InitialDelay: 3 * time.Second, MaxDelay: 10 * time.Second}}}}, Validate: func(value any) error {
		c := value.(testDemoConfig)
		if ref := c.Demo.SecretCredential.Reference; ref != "" {
			if scheme, rest, ok := strings.Cut(ref, "://"); !ok || scheme == "" || rest == "" {
				return fmt.Errorf("invalid demo.secret_credential.reference: must be scheme://reference")
			}
		}
		policy := c.Retry.Demo.FlakyDependency
		if policy.MaxAttempts < 1 {
			return fmt.Errorf("invalid retry.demo.flaky_dependency.max_attempts")
		}
		if policy.InitialDelay < 0 || policy.MaxDelay < 0 || (policy.MaxAttempts > 1 && (policy.InitialDelay == 0 || policy.MaxDelay < policy.InitialDelay)) {
			return fmt.Errorf("invalid retry.demo.flaky_dependency delays")
		}
		if math.IsNaN(policy.Jitter) || policy.Jitter < 0 || policy.Jitter > 1 {
			return fmt.Errorf("invalid retry.demo.flaky_dependency.jitter")
		}
		for _, timeout := range []*time.Duration{c.Timeouts.Demo.StructuredOutput, c.Timeouts.Demo.NotFound, c.Timeouts.Demo.SecretCredential} {
			if timeout != nil && *timeout <= 0 {
				return fmt.Errorf("demo timeout must be positive")
			}
		}
		return nil
	}}
}

func loadWithDemo(files ConfigFiles, environment []string, flags map[string]FlagValue) (LoadResult, error) {
	return load(files, environment, flags, demoTestRegistration())
}
