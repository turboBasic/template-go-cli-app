package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestLoadFilesPrecedenceAndSources(t *testing.T) {
	root := t.TempDir()
	userPath := filepath.Join(root, "user.toml")
	projectPath := filepath.Join(root, "project.toml")
	if err := os.WriteFile(userPath, []byte("output = 'json'\n[log]\nlevel = 'info'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectPath, []byte("[log]\nlevel = 'error'\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadFiles(ConfigFiles{User: userPath, Project: projectPath})
	if err != nil {
		t.Fatal(err)
	}
	if want := (Config{Log: LogConfig{Level: "error", File: FileLogConfig{Level: "debug"}}, Output: "json", Color: "auto", Quiet: false, Timeouts: TimeoutConfig{Cleanup: 5 * time.Second}, Retry: Defaults().Retry}); got.Config != want {
		t.Fatalf("Config = %+v, want %+v", got.Config, want)
	}
	for key, want := range map[string]Source{
		"log.level":      Source(projectPath),
		"log.file.level": defaultSource,
		"log.file.path":  defaultSource,
		"output":         Source(userPath),
		"color":          defaultSource,
		"quiet":          defaultSource,
	} {
		if got.Sources[key] != want {
			t.Errorf("source for %q = %q, want %q", key, got.Sources[key], want)
		}
	}
}

func TestMergeLayerRecursivelyAndReplacesArrays(t *testing.T) {
	values := map[string]any{
		"server": map[string]any{"host": "localhost", "port": int64(80)},
		"tags":   []any{"base"},
		"name":   "base",
	}
	sources := map[string]Source{
		"server.host": defaultSource,
		"server.port": defaultSource,
		"tags":        defaultSource,
		"name":        defaultSource,
	}
	mergeLayer(values, map[string]any{
		"server": map[string]any{"port": int64(90)},
		"tags":   []any{"user", "only"},
		"name":   "user",
	}, Source("user.toml"), sources)
	mergeLayer(values, map[string]any{
		"server": map[string]any{"host": "production"},
		"tags":   []any{"project"},
	}, Source("project.toml"), sources)

	want := map[string]any{
		"server": map[string]any{"host": "production", "port": int64(90)},
		"tags":   []any{"project"},
		"name":   "user",
	}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("merged values = %#v, want %#v", values, want)
	}
	wantSources := map[string]Source{
		"server.host": "project.toml",
		"server.port": "user.toml",
		"tags":        "project.toml",
		"name":        "user.toml",
	}
	if !reflect.DeepEqual(sources, wantSources) {
		t.Fatalf("sources = %#v, want %#v", sources, wantSources)
	}
}

func TestEveryEffectiveLeafHasASource(t *testing.T) {
	root := t.TempDir()
	userPath := filepath.Join(root, "user.toml")
	data := "[timeouts.demo]\nstructured_output = '3s'\n[retry.demo.flaky_dependency]\nmax_attempts = 2\ninitial_delay = '1s'\nmax_delay = '2s'\njitter = 0.5\n"
	if err := os.WriteFile(userPath, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadFiles(ConfigFiles{User: userPath})
	if err != nil {
		t.Fatal(err)
	}
	if got.Sources["timeouts.cleanup"] != defaultSource {
		t.Errorf("source for timeouts.cleanup = %q, want %q", got.Sources["timeouts.cleanup"], defaultSource)
	}
	if got.Config.Timeouts.Cleanup != 5*time.Second {
		t.Errorf("Timeouts.Cleanup = %s, want 5s", got.Config.Timeouts.Cleanup)
	}
	if got.Sources["timeouts.demo.structured_output"] != Source(userPath) || got.Sources["retry.demo.flaky_dependency.max_attempts"] != Source(userPath) {
		t.Errorf("user-file sources = %q, %q, want %q", got.Sources["timeouts.demo.structured_output"], got.Sources["retry.demo.flaky_dependency.max_attempts"], userPath)
	}
	for _, leaf := range leaves(got.values) {
		if got.Sources[leaf.key] == "" {
			t.Errorf("leaf %q has no source", leaf.key)
		}
	}
}

// Every supported setting must be listed in defaultValues, and the Config fields must match the schema keys;
// a koanf tag that differs from its schema key would leave an override reported but never applied.
func TestSchemaKeysAreListedAndMatchConfigTags(t *testing.T) {
	listed := map[string]bool{}
	for _, leaf := range leaves(defaultValues()) {
		listed[leaf.key] = true
	}
	tagged := map[string]bool{}
	var walk func(reflect.Type, string)
	walk = func(typ reflect.Type, prefix string) {
		for field := range typ.Fields() {
			key := prefix + field.Tag.Get("koanf")
			if field.Type.Kind() == reflect.Struct {
				walk(field.Type, key+".")
				continue
			}
			tagged[key] = true
		}
	}
	walk(reflect.TypeFor[Config](), "")
	for key := range productionSchema() {
		if !listed[key] {
			t.Errorf("schema key %q has no entry in defaultValues", key)
		}
		if !tagged[key] {
			t.Errorf("schema key %q has no Config field", key)
		}
		delete(listed, key)
		delete(tagged, key)
	}
	for key := range listed {
		t.Errorf("defaultValues key %q is not in the schema", key)
	}
	for key := range tagged {
		t.Errorf("Config field %q is not in the schema", key)
	}
}
