package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDecodeTOML(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    Config
		wantApp any
		wantIn  string
	}{
		{
			name: "valid settings",
			data: "output = 'json'\nquiet = true\n[log]\nlevel = 'debug'\n[log.file]\nlevel = 'error'\npath = 'logs/app.jsonl'\n",
			want: withDefaults(func(c *Config) {
				c.Log.Level, c.Log.File.Level, c.Log.File.Path, c.Output, c.Quiet = "debug", "error", "logs/app.jsonl", "json", true
			}),
		},
		{
			name:    "valid operation timeouts",
			data:    "[timeouts]\ncleanup = '8s'\n[timeouts.demo]\nstructured_output = '30s'\nnot_found = '1m'\nsecret_credential = '45s'\n",
			want:    withDefaults(func(c *Config) { c.Timeouts.Cleanup = 8 * time.Second }),
			wantApp: testDemoConfig{Timeouts: testTimeoutRoot{Demo: testDemoTimeouts{StructuredOutput: durationPtr(30 * time.Second), NotFound: durationPtr(time.Minute), SecretCredential: durationPtr(45 * time.Second)}}, Retry: demoTestRegistration().Defaults.(testDemoConfig).Retry},
		},
		{
			name: "valid status retry policy",
			data: "[retry.demo.flaky_dependency]\nmax_attempts = 3\ninitial_delay = '100ms'\nmax_delay = '1s'\njitter = 0.25\n",
			want: Defaults(),
			wantApp: func() testDemoConfig {
				c := demoTestRegistration().Defaults.(testDemoConfig)
				c.Retry.Demo.FlakyDependency = testRetryPolicy{MaxAttempts: 3, InitialDelay: 100 * time.Millisecond, MaxDelay: time.Second, Jitter: .25}
				return c
			}(),
		},
		{name: "malformed TOML", data: "log-level = [", wantIn: "settings.toml"},
		{name: "unknown key", data: "log-levels = 'warn'\n", wantIn: `unknown key "log-levels"`},
		{name: "unknown nested key", data: "[log]\nlevels = 'warn'\n", wantIn: `unknown key "log.levels"`},
		{name: "empty known table", data: "[log]\n", want: Defaults()},
		{name: "empty known nested table", data: "[log.file]\n", want: Defaults()},
		{name: "empty retry table keeps the default policy", data: "[retry.demo.flaky_dependency]\n", want: Defaults(), wantApp: demoTestRegistration().Defaults},
		{name: "unknown empty table", data: "[typo]\n", wantIn: `unknown key "typo"`},
		{name: "unknown empty nested table", data: "[typo.sub]\n", wantIn: `unknown key "typo"`},
		{name: "retired empty table", data: "[log.console]\n", wantIn: `unknown key "log.console"`},
		{name: "table at scalar key", data: "[output]\n", wantIn: `invalid type for key "output"`},
		{name: "inline table at scalar key", data: "output = {}\n", wantIn: `invalid type for key "output"`},
		{name: "table at duration key", data: "[timeouts.cleanup]\n", wantIn: `invalid type for key "timeouts.cleanup"`},
		{name: "wrong nested string type", data: "[log]\nlevel = 3\n", wantIn: `invalid type for key "log.level"`},
		{name: "wrong string type", data: "output = 3\n", wantIn: `invalid type for key "output"`},
		{name: "wrong boolean type", data: "quiet = 'yes'\n", wantIn: `invalid type for key "quiet"`},
		{name: "invalid value", data: "[log]\nlevel = 'trace'\n", wantIn: "invalid log level"},
		{name: "invalid duration syntax", data: "[timeouts.demo]\nstructured_output = 'soon'\n", wantIn: "invalid duration"},
		{name: "zero operation duration", data: "[timeouts.demo]\nstructured_output = '0s'\n", wantIn: "must be positive"},
		{name: "negative operation duration", data: "[timeouts.demo]\nsecret_credential = '-1s'\n", wantIn: "must be positive"},
		{name: "zero cleanup duration", data: "[timeouts]\ncleanup = '0s'\n", wantIn: "must be positive"},
		{name: "retry policy zero initial delay", data: "[retry.demo.flaky_dependency]\nmax_attempts = 3\ninitial_delay = '0s'\n", wantIn: "retry.demo.flaky_dependency delays"},
		{name: "retry policy max delay below initial", data: "[retry.demo.flaky_dependency]\nmax_attempts = 2\ninitial_delay = '1s'\nmax_delay = '1ms'\n", wantIn: "retry.demo.flaky_dependency delays"},
		{name: "retry policy invalid attempts", data: "[retry.demo.flaky_dependency]\nmax_attempts = 0\n", wantIn: "max_attempts"},
		{name: "retry policy invalid jitter", data: "[retry.demo.flaky_dependency]\nmax_attempts = 2\ninitial_delay = '1ms'\nmax_delay = '1s'\njitter = 1.5\n", wantIn: "jitter"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, appSettings, err := DecodeTOMLWithSettings("settings.toml", []byte(test.data), demoTestRegistration())
			if test.wantIn != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantIn) || !strings.Contains(err.Error(), "settings.toml") {
					t.Fatalf("DecodeTOML() error = %v, want it to contain %q and source", err, test.wantIn)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeTOML() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("DecodeTOML() = %+v, want %+v", got, test.want)
			}
			if test.wantApp != nil && !reflect.DeepEqual(appSettings, test.wantApp) {
				t.Fatalf("application settings = %+v, want %+v", appSettings, test.wantApp)
			}
		})
	}
}
