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
	result, err := mergeFiles(files)
	if err != nil {
		return LoadResult{}, err
	}
	for _, entry := range environment {
		name, value, ok := strings.Cut(entry, "=")
		if !ok || !strings.HasPrefix(name, "CLI_APP_") {
			continue
		}
		key := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(name, "CLI_APP_"), "_", "-"))
		parsed, known, err := parseOverlayValue(key, value)
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
		parsed, known, err := parseFlagValue(key, flag.Value)
		if !known {
			continue
		}
		if err != nil {
			return LoadResult{}, fmt.Errorf("read flag %s: %w", flagSource(key, flag), err)
		}
		mergeLayer(result.values, map[string]any{key: parsed}, Source(flagSource(key, flag)), result.Sources)
	}
	result.Config, err = decodeConfig(result.values)
	if err != nil {
		return LoadResult{}, fmt.Errorf("decode effective configuration: %w", err)
	}
	return result, nil
}

func parseOverlayValue(key, value string) (any, bool, error) {
	switch key {
	case "log-level", "output", "color":
		return value, true, nil
	case "quiet":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return nil, true, fmt.Errorf("invalid boolean %q", value)
		}
		return parsed, true, nil
	default:
		return nil, false, nil
	}
}

func parseFlagValue(key string, value any) (any, bool, error) {
	switch key {
	case "log-level", "output", "color":
		parsed, ok := value.(string)
		if !ok {
			return nil, true, fmt.Errorf("expected string, got %T", value)
		}
		return parsed, true, nil
	case "quiet":
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
