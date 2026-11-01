package config

import (
	"strings"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	if got, want := Defaults(), (Config{
		LogLevel: "warn",
		Log:      LogConfig{Console: ConsoleLogConfig{Level: "warn"}, File: FileLogConfig{Level: "debug"}},
		Output:   "text",
		Color:    "auto",
		Timeouts: TimeoutConfig{Cleanup: 5 * time.Second},
	}); got != want {
		t.Fatalf("Defaults() = %+v, want %+v", got, want)
	}
}

func durationPtr(value time.Duration) *time.Duration { return &value }

func TestTimeoutDefaultsAndValidation(t *testing.T) {
	defaults := Defaults()
	if defaults.Timeouts.Cleanup != 5*time.Second {
		t.Fatalf("cleanup timeout = %s, want 5s", defaults.Timeouts.Cleanup)
	}
	if defaults.Timeouts.Resource.List != nil || defaults.Timeouts.Resource.Status != nil || defaults.Timeouts.Resource.Apply != nil {
		t.Fatalf("operation timeouts = %+v, want disabled defaults", defaults.Timeouts.Resource)
	}
	for _, test := range []struct {
		name string
		edit func(*Config)
	}{
		{name: "zero operation timeout", edit: func(c *Config) { timeout := time.Duration(0); c.Timeouts.Resource.List = &timeout }},
		{name: "negative operation timeout", edit: func(c *Config) { timeout := -time.Second; c.Timeouts.Resource.Apply = &timeout }},
		{name: "zero cleanup timeout", edit: func(c *Config) { c.Timeouts.Cleanup = 0 }},
		{name: "negative cleanup timeout", edit: func(c *Config) { c.Timeouts.Cleanup = -time.Second }},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := Defaults()
			test.edit(&config)
			if err := config.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want non-positive timeout error")
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Config)
		want string
	}{
		{name: "valid defaults"},
		{name: "invalid console log level", edit: func(c *Config) { c.Log.Console.Level = "trace" }, want: "invalid console log level"},
		{name: "invalid file log level", edit: func(c *Config) { c.Log.File.Level = "trace" }, want: "invalid file log level"},
		{name: "invalid output", edit: func(c *Config) { c.Output = "yaml" }, want: "invalid output"},
		{name: "invalid color", edit: func(c *Config) { c.Color = "sometimes" }, want: "invalid color"},
		{name: "json with forced color", edit: func(c *Config) { c.Output, c.Color = "json", "always" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := Defaults()
			if test.edit != nil {
				test.edit(&config)
			}
			err := config.Validate()
			if test.want == "" && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)) {
				t.Fatalf("Validate() error = %v, want it to contain %q", err, test.want)
			}
		})
	}
}
