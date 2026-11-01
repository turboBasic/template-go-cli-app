package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// FlagValue describes a registered flag's current value and changed state.
type FlagValue struct {
	Value   any
	Changed bool
	Source  string
}

// Load applies environment variables and changed flags above the discovered files.
func Load(files ConfigFiles, flags map[string]FlagValue) (LoadResult, error) {
	return load(files, os.Environ(), flags)
}

func load(files ConfigFiles, environment []string, flags map[string]FlagValue) (LoadResult, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return LoadResult{}, fmt.Errorf("find working directory: %w", err)
	}
	return loadWithSchema(files, environment, flags, productionSchema(), workingDir)
}

func loadWithSchema(files ConfigFiles, environment []string, flags map[string]FlagValue, schema map[string]valueType, workingDir string) (LoadResult, error) {
	result, err := mergeFilesWithSchema(files, schema)
	if err != nil {
		return LoadResult{}, err
	}
	for _, entry := range environment {
		name, value, ok := strings.Cut(entry, "=")
		if !ok || !strings.HasPrefix(name, "CLI_APP_") {
			continue
		}
		key := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(name, "CLI_APP_"), "_", "-"))
		parsed, known, err := parseOverlayValue(key, value, schema)
		if !known {
			continue
		}
		if err != nil {
			return LoadResult{}, fmt.Errorf("read environment variable %s: %w", name, err)
		}
		mergeLayer(result.values, map[string]any{key: parsed}, Source(name), result.Sources)
	}
	for key, flag := range flags {
		if !flag.Changed {
			continue
		}
		parsed, known, err := parseFlagValue(key, flag.Value, schema)
		if !known {
			continue
		}
		if err != nil {
			return LoadResult{}, fmt.Errorf("read flag %s: %w", flagSource(key, flag), err)
		}
		mergeLayer(result.values, map[string]any{key: parsed}, Source(flagSource(key, flag)), result.Sources)
	}
	if err := resolvePaths(result.values, result.Sources, workingDir, schema); err != nil {
		return LoadResult{}, err
	}
	result.Config, err = decodeConfig(result.values)
	if err != nil {
		return LoadResult{}, fmt.Errorf("decode effective configuration: %w", err)
	}
	return result, nil
}

func parseOverlayValue(key, value string, schema map[string]valueType) (any, bool, error) {
	kind, known := schema[key]
	if !known {
		return nil, false, nil
	}
	switch kind {
	case stringValue, pathValue:
		return value, true, nil
	case booleanValue:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return nil, true, fmt.Errorf("invalid boolean %q", value)
		}
		return parsed, true, nil
	default:
		return nil, false, nil
	}
}

func parseFlagValue(key string, value any, schema map[string]valueType) (any, bool, error) {
	kind, known := schema[key]
	if !known {
		return nil, false, nil
	}
	switch kind {
	case stringValue, pathValue:
		parsed, ok := value.(string)
		if !ok {
			return nil, true, fmt.Errorf("expected string, got %T", value)
		}
		return parsed, true, nil
	case booleanValue:
		parsed, ok := value.(bool)
		if !ok {
			return nil, true, fmt.Errorf("expected boolean, got %T", value)
		}
		return parsed, true, nil
	default:
		return nil, false, nil
	}
}

func flagSource(key string, flag FlagValue) string {
	if flag.Source != "" {
		return flag.Source
	}
	return "--" + key
}
