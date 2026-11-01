package config

import (
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	if got, want := Defaults(), (Config{
		LogLevel: "warn",
		Log:      LogConfig{Console: ConsoleLogConfig{Level: "warn"}, File: FileLogConfig{Level: "debug"}},
		Output:   "text",
		Color:    "auto",
	}); got != want {
		t.Fatalf("Defaults() = %+v, want %+v", got, want)
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
		{name: "json with forced color", edit: func(c *Config) { c.Output, c.Color = "json", "always" }, want: "cannot be used"},
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
