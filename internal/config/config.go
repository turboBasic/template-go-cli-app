package config

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Config contains the supported Stage 1 application settings.
type Config struct {
	Log      LogConfig      `koanf:"log"`
	Output   string         `koanf:"output"`
	Color    string         `koanf:"color"`
	Quiet    bool           `koanf:"quiet"`
	Timeouts TimeoutConfig  `koanf:"timeouts"`
	Retry    RetryConfig    `koanf:"retry"`
	Resource ResourceConfig `koanf:"resource"`
}

// ResourceConfig holds settings for resource operations.
type ResourceConfig struct {
	// Credential is a secret reference such as op://vault/item/field, never the secret itself.
	Credential string `koanf:"credential"`
}

// RetryConfig holds opt-in policies for individually approved operations.
type RetryConfig struct {
	Resource ResourceRetryConfig `koanf:"resource"`
}

// ResourceRetryConfig holds retry policies for resource operations.
type ResourceRetryConfig struct {
	Status *RetryPolicyConfig `koanf:"status"`
}

// RetryPolicyConfig is a bounded retry policy for one configured operation.
type RetryPolicyConfig struct {
	MaxAttempts  int           `koanf:"max_attempts"`
	InitialDelay time.Duration `koanf:"initial_delay"`
	MaxDelay     time.Duration `koanf:"max_delay"`
	Jitter       float64       `koanf:"jitter"`
}

// TimeoutConfig defines per-operation timeouts and the dependency cleanup bound.
type TimeoutConfig struct {
	Cleanup  time.Duration    `koanf:"cleanup"`
	Resource ResourceTimeouts `koanf:"resource"`
}

// ResourceTimeouts configures timeouts for resource use cases.
type ResourceTimeouts struct {
	List   *time.Duration `koanf:"list"`
	Status *time.Duration `koanf:"status"`
	Apply  *time.Duration `koanf:"apply"`
}

// LogConfig contains the settings for local log destinations.
type LogConfig struct {
	// Level is the human-readable stderr level, set by --log-level.
	Level string        `koanf:"level"`
	File  FileLogConfig `koanf:"file"`
}

// FileLogConfig configures optional JSON Lines file logging.
type FileLogConfig struct {
	Path  string `koanf:"path"`
	Level string `koanf:"level"`
}

// Defaults returns the settings used when no source overrides them.
func Defaults() Config {
	return Config{
		Log:      LogConfig{Level: "warn", File: FileLogConfig{Level: "debug"}},
		Output:   "text",
		Color:    "auto",
		Timeouts: TimeoutConfig{Cleanup: 5 * time.Second},
	}
}

// Validate checks that the current settings are supported together.
func (c Config) Validate() error {
	if !oneOf(c.Log.Level, "debug", "info", "warn", "error") {
		return fmt.Errorf("invalid log level %q: must be debug, info, warn, or error", c.Log.Level)
	}
	if !oneOf(c.Log.File.Level, "debug", "info", "warn", "error") {
		return fmt.Errorf("invalid file log level %q: must be debug, info, warn, or error", c.Log.File.Level)
	}
	if !oneOf(c.Output, "text", "json") {
		return fmt.Errorf("invalid output %q: must be text or json", c.Output)
	}
	if !oneOf(c.Color, "auto", "always", "never") {
		return fmt.Errorf("invalid color %q: must be auto, always, or never", c.Color)
	}
	if c.Timeouts.Cleanup <= 0 {
		return fmt.Errorf("invalid timeouts.cleanup %s: must be positive", c.Timeouts.Cleanup)
	}
	for _, setting := range []struct {
		key     string
		timeout *time.Duration
	}{
		{key: "timeouts.resource.list", timeout: c.Timeouts.Resource.List},
		{key: "timeouts.resource.status", timeout: c.Timeouts.Resource.Status},
		{key: "timeouts.resource.apply", timeout: c.Timeouts.Resource.Apply},
	} {
		if setting.timeout != nil && *setting.timeout <= 0 {
			return fmt.Errorf("invalid %s %s: must be positive", setting.key, *setting.timeout)
		}
	}
	if c.Retry.Resource.Status != nil {
		policy := c.Retry.Resource.Status
		if policy.MaxAttempts < 1 {
			return fmt.Errorf("invalid retry.resource.status.max_attempts: must be positive")
		}
		if policy.InitialDelay < 0 || policy.MaxDelay < 0 || (policy.MaxAttempts > 1 && (policy.InitialDelay == 0 || policy.MaxDelay < policy.InitialDelay)) {
			return fmt.Errorf("invalid retry.resource.status delays: require 0 < initial <= max for multiple attempts")
		}
		if math.IsNaN(policy.Jitter) || policy.Jitter < 0 || policy.Jitter > 1 {
			return fmt.Errorf("invalid retry.resource.status.jitter: must be between 0 and 1")
		}
	}
	if ref := c.Resource.Credential; ref != "" {
		if scheme, rest, ok := strings.Cut(ref, "://"); !ok || scheme == "" || rest == "" {
			return fmt.Errorf("invalid resource.credential: must be a scheme://reference, not the secret itself")
		}
	}
	return nil
}

func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
