package config

import (
	"fmt"
	"maps"
	"os"
	"slices"
)

// Source identifies the configuration layer and origin that supplied a value.
type Source struct {
	Kind   SourceKind
	Origin string
}

func defaultSource() Source { return Source{Kind: KindDefault} }

// LoadResult contains typed settings and the source of every effective leaf.
type LoadResult struct {
	Config  Config
	Sources map[string]Source
	values  map[string]any
}

func mergeFiles(files ConfigFiles) (LoadResult, error) {
	values := defaultValues()
	sources := make(map[string]Source)
	for _, leaf := range leaves(values) {
		sources[leaf.key] = defaultSource()
	}
	for _, file := range []struct {
		path string
		kind SourceKind
	}{{files.User, KindUserFile}, {files.Project, KindProjectFile}} {
		path := file.path
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return LoadResult{}, fmt.Errorf("read configuration %s: %w", path, err)
		}
		layer, err := parseTOML(path, data)
		if err != nil {
			return LoadResult{}, err
		}
		mergeLayer(values, layer, Source{Kind: file.kind, Origin: path}, sources)
	}
	return LoadResult{Sources: sources, values: values}, nil
}

type leaf struct {
	key   string
	value any
}

// leaves walks nested maps depth-first in key order and returns each non-map value under its dotted key.
func leaves(values map[string]any) []leaf {
	var result []leaf
	var walk func(prefix string, group map[string]any)
	walk = func(prefix string, group map[string]any) {
		for _, key := range slices.Sorted(maps.Keys(group)) {
			path := prefix + key
			if nested, ok := group[key].(map[string]any); ok {
				walk(path+".", nested)
				continue
			}
			result = append(result, leaf{key: path, value: group[key]})
		}
	}
	walk("", values)
	return result
}

func mergeLayer(values, layer map[string]any, source Source, sources map[string]Source) {
	mergeMap(values, layer, "", source, sources)
}

func mergeMap(values, layer map[string]any, prefix string, source Source, sources map[string]Source) {
	for key, incoming := range layer {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		if incomingMap, ok := incoming.(map[string]any); ok {
			currentMap, ok := values[key].(map[string]any)
			if !ok {
				deleteSourceTree(sources, path)
				currentMap = make(map[string]any, len(incomingMap))
				values[key] = currentMap
			}
			delete(sources, path)
			mergeMap(currentMap, incomingMap, path, source, sources)
			continue
		}
		deleteSourceTree(sources, path)
		values[key] = incoming
		sources[path] = source
	}
}

func deleteSourceTree(sources map[string]Source, path string) {
	for key := range sources {
		if key == path || len(key) > len(path) && key[:len(path)+1] == path+"." {
			delete(sources, key)
		}
	}
}
