package config

import (
	"fmt"
	"time"
)

// Config contains the supported Stage 1 application settings.
type Config struct {
	Log      LogConfig     `koanf:"log"`
	Output   string        `koanf:"output"`
	Color    string        `koanf:"color"`
	Quiet    bool          `koanf:"quiet"`
	Timeouts TimeoutConfig `koanf:"timeouts"`
	Retry    RetryConfig   `koanf:"retry"`
	Demo     DemoConfig    `koanf:"demo"`
}

// TimeoutConfig defines per-operation timeouts and the dependency cleanup bound.
type TimeoutConfig struct {
	Cleanup time.Duration `koanf:"cleanup"`
	Demo    DemoTimeouts  `koanf:"demo"`
}

// LogConfig contains the settings for local log destinations.
type LogConfig struct {
	// Level is the human-readable stderr level, set by --log-level.
	Level string        `koanf:"level"`
	File  FileLogConfig `koanf:"file"`
}

// FileLogConfig configures optional JSON Lines file logging.
type FileLogConfig struct {
	Path  string `koanf:"path" config:"path"`
	Level string `koanf:"level"`
}

// Defaults returns the settings used when no source overrides them.
func Defaults() Config {
	return Config{
		Log:      LogConfig{Level: "warn", File: FileLogConfig{Level: "debug"}},
		Output:   "text",
		Color:    "auto",
		Timeouts: TimeoutConfig{Cleanup: 5 * time.Second},
		Retry:    demoRetryDefaults(),
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
	return c.validateDemo()
}

func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
