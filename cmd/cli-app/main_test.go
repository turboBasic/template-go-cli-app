package main

import (
	"runtime/debug"
	"testing"
)

func TestBuildInfoIncludesEmbeddedVCSMetadata(t *testing.T) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Skip("Go build metadata is unavailable")
	}
	var revision, buildDate string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.time":
			buildDate = setting.Value
		}
	}
	got := buildInfo()
	if got.Version != version {
		t.Errorf("Version = %q, want %q", got.Version, version)
	}
	if revision != "" && got.Commit != revision {
		t.Errorf("Commit = %q, want %q", got.Commit, revision)
	}
	if buildDate != "" && got.BuildDate != buildDate {
		t.Errorf("BuildDate = %q, want %q", got.BuildDate, buildDate)
	}
}
