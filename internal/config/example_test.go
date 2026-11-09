package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/cli"
	"github.com/turboBasic/template-go-cli-app/internal/config"
)

func TestExampleFileMatchesRegisteredSchemasAndDefaults(t *testing.T) {
	paths, err := filepath.Glob("../../.*.example.toml")
	if err != nil || len(paths) != 1 {
		t.Fatalf("want exactly one root .*.example.toml, got %v (err %v)", paths, err)
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	var uncommented []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "# ") && line != "#" {
			line = line[1:]
		}
		uncommented = append(uncommented, line)
	}
	text := strings.Join(uncommented, "\n")
	path := filepath.Join(t.TempDir(), "example.toml")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	settings := cli.DemoApplicationSettings()
	loaded, err := config.LoadWithSettings(config.ConfigFiles{Project: path}, nil, settings)
	if err != nil {
		t.Fatal(err)
	}
	keys := make(map[string]bool)
	for _, entry := range loaded.Entries() {
		keys[entry.Key] = true
	}
	var prefix string
	for _, line := range uncommented {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			prefix = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, _, ok := strings.Cut(line, " = ")
		if ok {
			if prefix != "" {
				key = prefix + "." + key
			}
			if !keys[key] {
				t.Errorf("%s is absent from registered config schema", key)
			}
		}
	}
	defaults, err := config.LoadWithSettings(config.ConfigFiles{}, nil, settings)
	if err != nil {
		t.Fatal(err)
	}
	gotApp := loaded.Application.(cli.DemoConfig)
	gotApp.Demo.SecretCredential.Reference = ""
	gotApp.Timeouts.Demo.StructuredOutput = nil
	gotApp.Timeouts.Demo.NotFound = nil
	gotApp.Timeouts.Demo.SecretCredential = nil
	gotShared := loaded.Config
	gotShared.Log.File.Path = ""
	if !reflect.DeepEqual(gotShared, defaults.Config) {
		t.Errorf("example shared defaults differ: got %+v want %+v", gotShared, defaults.Config)
	}
	if !reflect.DeepEqual(gotApp, defaults.Application) {
		t.Errorf("example application defaults differ: got %+v want %+v", gotApp, defaults.Application)
	}
}
