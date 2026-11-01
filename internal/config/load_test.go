package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadFilesPrecedenceAndSources(t *testing.T) {
	root := t.TempDir()
	userPath := filepath.Join(root, "user.toml")
	projectPath := filepath.Join(root, "project.toml")
	if err := os.WriteFile(userPath, []byte("log-level = 'info'\noutput = 'json'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectPath, []byte("log-level = 'error'\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadFiles(ConfigFiles{User: userPath, Project: projectPath})
	if err != nil {
		t.Fatal(err)
	}
	if want := (Config{LogLevel: "error", Output: "json", Color: "auto", Quiet: false}); got.Config != want {
		t.Fatalf("Config = %+v, want %+v", got.Config, want)
	}
	for key, want := range map[string]Source{
		"log-level": Source(projectPath),
		"output":    Source(userPath),
		"color":     defaultSource,
		"quiet":     defaultSource,
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
