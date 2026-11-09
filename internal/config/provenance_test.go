package config

import (
	"path/filepath"
	"testing"
)

func TestEntriesClassifyEverySource(t *testing.T) {
	root := t.TempDir()
	files := ConfigFiles{User: filepath.Join(root, "user.toml"), Project: filepath.Join(root, "project.toml")}
	writeFile(t, files.User, "[log.file]\nlevel = 'info'\n[log]\nlevel = 'info'\n")
	writeFile(t, files.Project, "[log]\nlevel = 'error'\n")

	result, err := loadAt(files, []string{"CLI_APP_COLOR=never"}, map[string]FlagValue{
		"quiet": {Value: true, Changed: true, Source: "--quiet"},
	}, root)
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]Entry{}
	for _, entry := range result.Entries() {
		entries[entry.Key] = entry
	}

	for _, want := range []Entry{
		{Key: "log.file.level", Source: KindUserFile, Origin: files.User},
		{Key: "log.level", Source: KindProjectFile, Origin: files.Project},
		{Key: "color", Source: KindEnvironment, Origin: "CLI_APP_COLOR"},
		{Key: "quiet", Source: KindFlag, Origin: "--quiet"},
		{Key: "output", Source: KindDefault},
	} {
		got := entries[want.Key]
		if got.Source != want.Source || got.Origin != want.Origin {
			t.Errorf("%s = (%q, %q), want (%q, %q)", want.Key, got.Source, got.Origin, want.Source, want.Origin)
		}
	}
	if got := entries["timeouts.cleanup"].Value; got != "5s" {
		t.Errorf("timeouts.cleanup value = %#v, want \"5s\"", got)
	}
	if got, ok := entries["timeouts.demo.structured_output"]; !ok || got.Value != nil || got.Source != KindDefault {
		t.Errorf("timeouts.demo.structured_output = %+v, want an unset default entry", got)
	}
	jitter := entries["retry.demo.flaky_dependency.jitter"]
	if jitter.Value != 0.0 || jitter.Source != KindDefault {
		t.Errorf("default jitter = (%#v, %q), want (0, default)", jitter.Value, jitter.Source)
	}
}

func TestEntriesReportWinningLayerWhenFilePathsMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.toml")
	writeFile(t, path, "output = 'json'\n")
	result, err := loadAt(ConfigFiles{User: path, Project: path}, nil, nil, filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range result.Entries() {
		if entry.Key == "output" {
			if entry.Source != KindProjectFile || entry.Origin != path {
				t.Errorf("output source = (%q, %q), want project file %q", entry.Source, entry.Origin, path)
			}
			return
		}
	}
	t.Fatal("no output entry")
}

func TestEntriesUnsourcedLeafIsUnknown(t *testing.T) {
	result := LoadResult{values: map[string]any{"output": "text"}}
	got := result.Entries()
	if len(got) != 1 || got[0].Source != KindUnknown {
		t.Errorf("Entries() = %+v, want one unknown entry", got)
	}
}

func TestEntriesReportJitterOverrideWithItsSource(t *testing.T) {
	root := t.TempDir()
	result, err := loadAt(ConfigFiles{}, []string{"CLI_APP_RETRY_DEMO_FLAKY_DEPENDENCY_JITTER=0.2"}, nil, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range result.Entries() {
		if entry.Key != "retry.demo.flaky_dependency.jitter" {
			continue
		}
		if entry.Value != 0.2 || entry.Source != KindEnvironment || entry.Origin != "CLI_APP_RETRY_DEMO_FLAKY_DEPENDENCY_JITTER" {
			t.Errorf("jitter = %+v, want 0.2 from the environment", entry)
		}
		return
	}
	t.Error("no entry for retry.demo.flaky_dependency.jitter")
}
