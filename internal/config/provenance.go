package config

import "time"

// SourceKind names the layer category that supplied an effective value.
type SourceKind string

// The layers of the precedence order, lowest first.
const (
	KindDefault     SourceKind = "default"
	KindUserFile    SourceKind = "user file"
	KindProjectFile SourceKind = "project file"
	KindEnvironment SourceKind = "environment variable"
	KindFlag        SourceKind = "flag"
	// KindUnknown marks a leaf whose source matches no layer; it signals a loader bug, not a user error.
	KindUnknown SourceKind = "unknown"
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
func (r LoadResult) Entries() []Entry {
	found := leaves(r.values)
	entries := make([]Entry, 0, len(found))
	for _, leaf := range found {
		value := leaf.value
		if duration, ok := value.(time.Duration); ok {
			value = duration.String()
		}
		source := r.Sources[leaf.key]
		kind := source.Kind
		if kind == "" {
			kind = KindUnknown
		}
		entries = append(entries, Entry{Key: leaf.key, Value: value, Source: kind, Origin: source.Origin})
	}
	return entries
}
