package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDecodeTOML(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		want   Config
		wantIn string
	}{
		{
			name: "valid settings",
			data: "output = 'json'\nquiet = true\n[log.console]\nlevel = 'debug'\n[log.file]\nlevel = 'error'\npath = 'logs/app.jsonl'\n",
			want: Config{LogLevel: "debug", Log: LogConfig{Console: ConsoleLogConfig{Level: "debug"}, File: FileLogConfig{Level: "error", Path: "logs/app.jsonl"}}, Output: "json", Color: "auto", Quiet: true, Timeouts: TimeoutConfig{Cleanup: 5 * time.Second}},
		},
		{
			name: "valid operation timeouts",
			data: "[timeouts]\ncleanup = '8s'\n[timeouts.resource]\nlist = '30s'\nstatus = '1m'\napply = '45s'\n",
			want: Config{LogLevel: "warn", Log: LogConfig{Console: ConsoleLogConfig{Level: "warn"}, File: FileLogConfig{Level: "debug"}}, Output: "text", Color: "auto", Timeouts: TimeoutConfig{Cleanup: 8 * time.Second, Resource: ResourceTimeouts{List: durationPtr(30 * time.Second), Status: durationPtr(time.Minute), Apply: durationPtr(45 * time.Second)}}},
		},
		{name: "malformed TOML", data: "log-level = [", wantIn: "settings.toml"},
		{name: "unknown key", data: "log-levels = 'warn'\n", wantIn: `unknown key "log-levels"`},
		{name: "unknown nested key", data: "[log.console]\nlevels = 'warn'\n", wantIn: `unknown key "log.console.levels"`},
		{name: "wrong nested string type", data: "[log.console]\nlevel = 3\n", wantIn: `invalid type for key "log.console.level"`},
		{name: "wrong string type", data: "output = 3\n", wantIn: `invalid type for key "output"`},
		{name: "wrong boolean type", data: "quiet = 'yes'\n", wantIn: `invalid type for key "quiet"`},
		{name: "invalid value", data: "[log.console]\nlevel = 'trace'\n", wantIn: "invalid console log level"},
		{name: "invalid duration syntax", data: "[timeouts.resource]\nlist = 'soon'\n", wantIn: "invalid duration"},
		{name: "zero operation duration", data: "[timeouts.resource]\nlist = '0s'\n", wantIn: "must be positive"},
		{name: "negative operation duration", data: "[timeouts.resource]\napply = '-1s'\n", wantIn: "must be positive"},
		{name: "zero cleanup duration", data: "[timeouts]\ncleanup = '0s'\n", wantIn: "must be positive"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := DecodeTOML("settings.toml", []byte(test.data))
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
		})
	}
}
