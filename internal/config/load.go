package config

import (
	"fmt"
	"os"
)

// Source identifies the configuration layer that supplied an effective value.
type Source string

const defaultSource Source = "default"

// LoadResult contains typed settings and the source of every effective leaf.
type LoadResult struct {
	Config  Config
	Sources map[string]Source
	values  map[string]any
}

// LoadFiles merges defaults, the user file, and the selected project file.
func LoadFiles(files ConfigFiles) (LoadResult, error) {
	result, err := mergeFiles(files)
	if err != nil {
		return LoadResult{}, err
	}
	config, err := decodeConfig(result.values)
	if err != nil {
		return LoadResult{}, fmt.Errorf("decode merged configuration: %w", err)
	}
	result.Config = config
	return result, nil
}

func mergeFiles(files ConfigFiles) (LoadResult, error) {
	return mergeFilesWithSchema(files, productionSchema())
}

func mergeFilesWithSchema(files ConfigFiles, schema map[string]valueType) (LoadResult, error) {
	values := defaultValues()
	sources := map[string]Source{
		"log.console.level": defaultSource,
		"log.file.level":    defaultSource,
		"log.file.path":     defaultSource,
		"output":            defaultSource,
		"color":             defaultSource,
		"quiet":             defaultSource,
	}
	for _, path := range []string{files.User, files.Project} {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return LoadResult{}, fmt.Errorf("read configuration %s: %w", path, err)
		}
		layer, err := decodeTOMLWithSchema(path, data, schema)
		if err != nil {
			return LoadResult{}, err
		}
		mergeLayer(values, layer, Source(path), sources)
	}
	return LoadResult{Sources: sources, values: values}, nil
}

func defaultValues() map[string]any {
	config := Defaults()
	return map[string]any{
		"log": map[string]any{
			"console": map[string]any{"level": config.Log.Console.Level},
			"file":    map[string]any{"path": config.Log.File.Path, "level": config.Log.File.Level},
		},
		"output": config.Output,
		"color":  config.Color,
		"quiet":  config.Quiet,
	}
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
