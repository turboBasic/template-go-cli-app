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
		value, ok := values[key]
		if !ok {
			continue
		}
		path, ok := value.(string)
		if !ok {
			return fmt.Errorf("resolve configuration path %q: expected string, got %T", key, value)
		}
		if filepath.IsAbs(path) {
			continue
		}
		base, err := pathBase(sources[key], workingDir)
		if err != nil {
			return fmt.Errorf("resolve configuration path %q: %w", key, err)
		}
		values[key] = filepath.Clean(filepath.Join(base, path))
	}
	return nil
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
