package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type independentSettings struct {
	Widget independentWidget `koanf:"widget"`
}
type independentWidget struct {
	Name  string        `koanf:"name"`
	Delay time.Duration `koanf:"delay"`
	Mode  string        `koanf:"mode"`
	Alias string        `koanf:"alias"`
}

func TestRegisteredApplicationSettingsUseAllSourcesAndProvenance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	writeFile(t, path, "[widget]\nname = 'from-file'\ndelay = '2s'\n")
	registration := ApplicationSettings{Defaults: independentSettings{Widget: independentWidget{Name: "default", Delay: time.Second, Mode: "safe"}}, Validate: func(value any) error {
		if value.(independentSettings).Widget.Name == "" {
			return fmt.Errorf("widget.name is required")
		}
		return nil
	}}
	got, err := load(ConfigFiles{Project: path}, []string{"CLI_APP_WIDGET_NAME=from-env", "CLI_APP_WIDGET_ALIAS=env-alias"}, map[string]FlagValue{
		"widget-name": {Value: "from-flag", Changed: true, Source: "--widget-name"},
	}, registration)
	if err != nil {
		t.Fatal(err)
	}
	want := independentSettings{Widget: independentWidget{Name: "from-flag", Delay: 2 * time.Second, Mode: "safe", Alias: "env-alias"}}
	if !reflect.DeepEqual(got.Application, want) {
		t.Fatalf("application settings = %+v, want %+v", got.Application, want)
	}
	for key, wantSource := range map[string]Source{"widget.name": {Kind: KindFlag, Origin: "--widget-name"}, "widget.delay": {Kind: KindProjectFile, Origin: path}, "widget.alias": {Kind: KindEnvironment, Origin: "CLI_APP_WIDGET_ALIAS"}} {
		if got.Sources[key] != wantSource {
			t.Errorf("source %s = %+v, want %+v", key, got.Sources[key], wantSource)
		}
	}
	if got.Sources["widget.mode"].Kind != KindDefault {
		t.Errorf("widget.mode source = %+v, want default", got.Sources["widget.mode"])
	}
	if _, err := load(ConfigFiles{}, []string{"CLI_APP_WIDGET_NAME="}, nil, registration); err == nil || !strings.Contains(err.Error(), "widget.name is required") {
		t.Fatalf("application validation error = %v", err)
	}
	unknownPath := filepath.Join(t.TempDir(), "unknown.toml")
	writeFile(t, unknownPath, "[widget]\nunknown = 'rejected'\n")
	if _, err := load(ConfigFiles{Project: unknownPath}, nil, nil, registration); err == nil || !strings.Contains(err.Error(), `unknown key "widget.unknown"`) {
		t.Fatalf("unknown application key error = %v", err)
	}
}

func TestRegisteredSettingsRejectCollisionsAndSharedOnlyRejectsRemovedKeys(t *testing.T) {
	type collision struct {
		Output string `koanf:"output"`
	}
	if _, err := settingsSchema(Defaults(), collision{}); err == nil || !strings.Contains(err.Error(), "output") {
		t.Fatalf("schema collision error = %v", err)
	}
	type tableCollision struct {
		Log string `koanf:"log"`
	}
	if _, err := settingsSchema(Defaults(), tableCollision{}); err == nil || !strings.Contains(err.Error(), "log") {
		t.Fatalf("table schema collision error = %v", err)
	}
	type duplicate struct {
		First  string `koanf:"setting"`
		Second int    `koanf:"setting"`
	}
	if _, err := settingsSchema(Defaults(), duplicate{}); err == nil || !strings.Contains(err.Error(), "setting") {
		t.Fatalf("duplicate application key error = %v", err)
	}
	sharedPath := filepath.Join(t.TempDir(), "shared.toml")
	writeFile(t, sharedPath, "output = 'json'\n")
	shared, err := Load(ConfigFiles{Project: sharedPath}, nil)
	if err != nil || shared.Config.Output != "json" {
		t.Fatalf("shared-only load = %+v, %v", shared.Config, err)
	}
	demoPath := filepath.Join(t.TempDir(), "removed.toml")
	writeFile(t, demoPath, "[demo.secret_credential]\nreference = 'op://vault/item/field'\n")
	if _, err := Load(ConfigFiles{Project: demoPath}, nil); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("shared-only demo key error = %v, want unknown key", err)
	}
}

func TestLoadFilePrecedenceAndSources(t *testing.T) {
	root := t.TempDir()
	userPath := filepath.Join(root, "user.toml")
	projectPath := filepath.Join(root, "project.toml")
	if err := os.WriteFile(userPath, []byte("output = 'json'\n[log]\nlevel = 'info'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectPath, []byte("[log]\nlevel = 'error'\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := loadWithDemo(ConfigFiles{User: userPath, Project: projectPath}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := Defaults()
	want.Log.Level, want.Output = "error", "json"
	if got.Config != want {
		t.Fatalf("Config = %+v, want %+v", got.Config, want)
	}
	for key, want := range map[string]Source{
		"log.level":      {Kind: KindProjectFile, Origin: projectPath},
		"log.file.level": defaultSource(),
		"log.file.path":  defaultSource(),
		"output":         {Kind: KindUserFile, Origin: userPath},
		"color":          defaultSource(),
		"quiet":          defaultSource(),
	} {
		if got.Sources[key] != want {
			t.Errorf("source for %q = %+v, want %+v", key, got.Sources[key], want)
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
		"server.host": defaultSource(),
		"server.port": defaultSource(),
		"tags":        defaultSource(),
		"name":        defaultSource(),
	}
	mergeLayer(values, map[string]any{
		"server": map[string]any{"port": int64(90)},
		"tags":   []any{"user", "only"},
		"name":   "user",
	}, Source{Kind: KindUserFile, Origin: "user.toml"}, sources)
	mergeLayer(values, map[string]any{
		"server": map[string]any{"host": "production"},
		"tags":   []any{"project"},
	}, Source{Kind: KindProjectFile, Origin: "project.toml"}, sources)

	want := map[string]any{
		"server": map[string]any{"host": "production", "port": int64(90)},
		"tags":   []any{"project"},
		"name":   "user",
	}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("merged values = %#v, want %#v", values, want)
	}
	wantSources := map[string]Source{
		"server.host": {Kind: KindProjectFile, Origin: "project.toml"},
		"server.port": {Kind: KindUserFile, Origin: "user.toml"},
		"tags":        {Kind: KindProjectFile, Origin: "project.toml"},
		"name":        {Kind: KindUserFile, Origin: "user.toml"},
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

	got, err := loadWithDemo(ConfigFiles{User: userPath}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sources["timeouts.cleanup"] != defaultSource() {
		t.Errorf("source for timeouts.cleanup = %+v, want %+v", got.Sources["timeouts.cleanup"], defaultSource())
	}
	if got.Config.Timeouts.Cleanup != 5*time.Second {
		t.Errorf("Timeouts.Cleanup = %s, want 5s", got.Config.Timeouts.Cleanup)
	}
	wantSource := Source{Kind: KindUserFile, Origin: userPath}
	if got.Sources["timeouts.demo.structured_output"] != wantSource || got.Sources["retry.demo.flaky_dependency.max_attempts"] != wantSource {
		t.Errorf("user-file sources = %+v, %+v, want %+v", got.Sources["timeouts.demo.structured_output"], got.Sources["retry.demo.flaky_dependency.max_attempts"], wantSource)
	}
	for _, leaf := range leaves(got.values) {
		if got.Sources[leaf.key].Kind == "" {
			t.Errorf("leaf %q has no source", leaf.key)
		}
	}
}
