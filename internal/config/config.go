package config

import "fmt"

// Config contains the supported Stage 1 application settings.
type Config struct {
	LogLevel string    `koanf:"-"` // Deprecated: use Log.Console.Level.
	Log      LogConfig `koanf:"log"`
	Output   string    `koanf:"output"`
	Color    string    `koanf:"color"`
	Quiet    bool      `koanf:"quiet"`
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
