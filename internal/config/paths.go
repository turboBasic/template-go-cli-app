package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

func resolvePaths(values map[string]any, sources map[string]Source, workingDir string, schema map[string]valueType) error {
	workingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}
	for key, kind := range schema {
		if kind != pathValue {
			continue
		}
		value, ok := nestedLookup(values, key)
		if !ok {
			continue
		}
		path, ok := value.(string)
		if !ok {
			return fmt.Errorf("resolve configuration path %q: expected string, got %T", key, value)
		}
		if path == "" {
			continue
		}
		if filepath.IsAbs(path) {
			continue
		}
		base, err := pathBase(sources[key], workingDir)
		if err != nil {
			return fmt.Errorf("resolve configuration path %q: %w", key, err)
		}
		nestedSet(values, key, filepath.Clean(filepath.Join(base, path)))
	}
	return nil
}

func nestedLookup(values map[string]any, key string) (any, bool) {
	parts := strings.Split(key, ".")
	var value any = values
	for _, part := range parts {
		group, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		value, ok = group[part]
		if !ok {
			return nil, false
		}
	}
	return value, true
}

func nestedSet(values map[string]any, key string, value any) {
	parts := strings.Split(key, ".")
	group := values
	for _, part := range parts[:len(parts)-1] {
		group = group[part].(map[string]any)
	}
	group[parts[len(parts)-1]] = value
}

func pathBase(source Source, workingDir string) (string, error) {
	identity := string(source)
	if identity == "" || identity == string(defaultSource) || strings.HasPrefix(identity, "CLI_APP_") || strings.HasPrefix(identity, "--") {
		return workingDir, nil
	}
	file, err := filepath.Abs(identity)
	if err != nil {
		return "", fmt.Errorf("resolve source %q: %w", identity, err)
	}
	return filepath.Dir(file), nil
}
