package config

import (
	"strings"
	"time"
)

// SourceKind names the layer category that supplied an effective value.
type SourceKind string

// The layers of the precedence order, lowest first.
const (
	KindDefault     SourceKind = "default"
	KindUserFile    SourceKind = "user file"
	KindProjectFile SourceKind = "project file"
	KindEnvironment SourceKind = "environment variable"
	KindFlag        SourceKind = "flag"
)

// Entry is one effective configuration leaf and where it came from.
type Entry struct {
	Key    string
	Value  any
	Source SourceKind
	// Origin is the file path, variable name or flag; empty for defaults.
	Origin string
}

// Entries lists every effective leaf sorted by key. Durations are rendered as their string form.
func (r LoadResult) Entries(files ConfigFiles) []Entry {
	found := leaves(r.values)
	entries := make([]Entry, 0, len(found))
	for _, leaf := range found {
		value := leaf.value
		if duration, ok := value.(time.Duration); ok {
			value = duration.String()
		}
		kind, origin := classify(r.Sources[leaf.key], files)
		entries = append(entries, Entry{Key: leaf.key, Value: value, Source: kind, Origin: origin})
	}
	return entries
}

// classify compares against the discovered files first, so a file path that starts with "--" is still a file.
func classify(source Source, files ConfigFiles) (SourceKind, string) {
	switch {
	case source == defaultSource:
		return KindDefault, ""
	case files.User != "" && string(source) == files.User:
		return KindUserFile, string(source)
	case files.Project != "" && string(source) == files.Project:
		return KindProjectFile, string(source)
	case strings.HasPrefix(string(source), "--"):
		return KindFlag, string(source)
	case strings.HasPrefix(string(source), "CLI_APP_"):
		return KindEnvironment, string(source)
	default:
		return KindProjectFile, string(source)
	}
}
