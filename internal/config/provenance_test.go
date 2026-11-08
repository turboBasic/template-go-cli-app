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

	result, err := loadWithSchema(files, []string{"CLI_APP_COLOR=never"}, map[string]FlagValue{
		"quiet": {Value: true, Changed: true, Source: "--quiet"},
	}, productionSchema(), root)
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
	if _, ok := entries["timeouts.resource.list"]; ok {
		t.Error("unset timeouts.resource.list has an entry")
	}
}

func TestEntriesFileStartingWithDashesIsStillAFile(t *testing.T) {
	kind, origin := classify("--odd.toml", ConfigFiles{Project: "--odd.toml"})
	if kind != KindProjectFile || origin != "--odd.toml" {
		t.Errorf("classify = (%q, %q), want project file", kind, origin)
	}
}

func TestEntriesUnsourcedLeafIsUnknown(t *testing.T) {
	result := LoadResult{values: map[string]any{"output": "text"}}
	got := result.Entries()
	if len(got) != 1 || got[0].Source != KindUnknown {
		t.Errorf("Entries() = %+v, want one unknown entry", got)
	}
}
