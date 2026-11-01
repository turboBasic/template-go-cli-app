package config

import (
	"fmt"
	"time"
)

// Config contains the supported Stage 1 application settings.
type Config struct {
	LogLevel string        `koanf:"-"` // Deprecated: use Log.Console.Level.
	Log      LogConfig     `koanf:"log"`
	Output   string        `koanf:"output"`
	Color    string        `koanf:"color"`
	Quiet    bool          `koanf:"quiet"`
	Timeouts TimeoutConfig `koanf:"timeouts"`
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
	Console ConsoleLogConfig `koanf:"console"`
	File    FileLogConfig    `koanf:"file"`
}

// ConsoleLogConfig configures human-readable stderr logging.
type ConsoleLogConfig struct {
	Level string `koanf:"level"`
}

// FileLogConfig configures optional JSON Lines file logging.
type FileLogConfig struct {
	Path  string `koanf:"path"`
	Level string `koanf:"level"`
}

// Defaults returns the settings used when no source overrides them.
func Defaults() Config {
	return Config{
		LogLevel: "warn",
		Log:      LogConfig{Console: ConsoleLogConfig{Level: "warn"}, File: FileLogConfig{Level: "debug"}},
		Output:   "text",
		Color:    "auto",
		Timeouts: TimeoutConfig{Cleanup: 5 * time.Second},
	}
}

// Validate checks that the current settings are supported together.
func (c Config) Validate() error {
	if !oneOf(c.Log.Console.Level, "debug", "info", "warn", "error") {
		return fmt.Errorf("invalid console log level %q: must be debug, info, warn, or error", c.Log.Console.Level)
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
