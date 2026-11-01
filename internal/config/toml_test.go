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
			data: "log-level = 'debug'\noutput = 'json'\nquiet = true\n",
			want: Config{LogLevel: "debug", Output: "json", Color: "auto", Quiet: true},
		},
		{name: "malformed TOML", data: "log-level = [", wantIn: "settings.toml"},
		{name: "unknown key", data: "log-level = 'debug'\nlog-levels = 'warn'\n", wantIn: `unknown key "log-levels"`},
		{name: "wrong string type", data: "output = 3\n", wantIn: `invalid type for key "output"`},
		{name: "wrong boolean type", data: "quiet = 'yes'\n", wantIn: `invalid type for key "quiet"`},
		{name: "invalid value", data: "color = 'sometimes'\n", wantIn: "invalid color"},
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
