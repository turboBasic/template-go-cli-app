package config

import (
	"strings"
	"testing"
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
			want: Config{LogLevel: "debug", Log: LogConfig{Console: ConsoleLogConfig{Level: "debug"}, File: FileLogConfig{Level: "error", Path: "logs/app.jsonl"}}, Output: "json", Color: "auto", Quiet: true},
		},
		{name: "malformed TOML", data: "log-level = [", wantIn: "settings.toml"},
		{name: "unknown key", data: "log-levels = 'warn'\n", wantIn: `unknown key "log-levels"`},
		{name: "unknown nested key", data: "[log.console]\nlevels = 'warn'\n", wantIn: `unknown key "log.console.levels"`},
		{name: "wrong nested string type", data: "[log.console]\nlevel = 3\n", wantIn: `invalid type for key "log.console.level"`},
		{name: "wrong string type", data: "output = 3\n", wantIn: `invalid type for key "output"`},
		{name: "wrong boolean type", data: "quiet = 'yes'\n", wantIn: `invalid type for key "quiet"`},
		{name: "invalid value", data: "[log.console]\nlevel = 'trace'\n", wantIn: "invalid console log level"},
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
			if got != test.want {
				t.Fatalf("DecodeTOML() = %+v, want %+v", got, test.want)
			}
		})
	}
}
