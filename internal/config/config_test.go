package config

import (
	"strings"
	"testing"
	"time"
)

// withDefaults returns Defaults() with one test's edits, so a case lists only what it changes.
func withDefaults(edit func(*Config)) Config {
	config := Defaults()
	edit(&config)
	return config
}

func durationPtr(value time.Duration) *time.Duration { return &value }

func TestTimeoutDefaultsAndValidation(t *testing.T) {
	defaults := Defaults()
	if defaults.Timeouts.Cleanup != 5*time.Second {
		t.Fatalf("cleanup timeout = %s, want 5s", defaults.Timeouts.Cleanup)
	}
	for _, test := range []struct {
		name string
		edit func(*Config)
	}{
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
		{name: "invalid log level", edit: func(c *Config) { c.Log.Level = "trace" }, want: "invalid log level"},
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
