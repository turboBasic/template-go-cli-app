package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// FlagValue describes a registered flag's current value and changed state.
type FlagValue struct {
	Value   any
	Changed bool
	Source  string
}

// Load applies environment variables and changed flags to shared settings.
func Load(files ConfigFiles, flags map[string]FlagValue) (LoadResult, error) {
	return load(files, os.Environ(), flags, ApplicationSettings{})
}

// LoadWithSettings applies the same sources to shared settings and one registered application schema.
func LoadWithSettings(files ConfigFiles, flags map[string]FlagValue, application ApplicationSettings) (LoadResult, error) {
	return load(files, os.Environ(), flags, application)
}

func load(files ConfigFiles, environment []string, flags map[string]FlagValue, application ApplicationSettings) (LoadResult, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return LoadResult{}, fmt.Errorf("find working directory: %w", err)
	}
	return loadAt(files, environment, flags, workingDir, application)
}

func loadAt(files ConfigFiles, environment []string, flags map[string]FlagValue, workingDir string, application ApplicationSettings) (LoadResult, error) {
	settings := []any{Defaults()}
	if application.Defaults != nil {
		settings = append(settings, application.Defaults)
	}
	schema, err := settingsSchema(settings...)
	if err != nil {
		return LoadResult{}, err
	}
	defaults := mergedDefaults(application)
	result, err := mergeFiles(files, schema, defaults)
	if err != nil {
		return LoadResult{}, err
	}
	for _, entry := range environment {
		name, value, ok := strings.Cut(entry, "=")
		if !ok || !strings.HasPrefix(name, "CLI_APP_") {
			continue
		}
		key := environmentKey(name, schema)
		parsed, known, err := parseOverlayValue(key, value, schema)
		if !known {
			continue
		}
		if err != nil {
			return LoadResult{}, fmt.Errorf("read environment variable %s: %w", name, err)
		}
		mergeLayer(result.values, nestedValue(key, parsed), Source{Kind: KindEnvironment, Origin: name}, result.Sources)
	}
	for key, flag := range flags {
		target := strings.ReplaceAll(key, "-", ".")
		if _, known := schema[target]; !known {
			return LoadResult{}, fmt.Errorf("flag %s has no configuration key %q", flagSource(key, flag), target)
		}
		if !flag.Changed {
			continue
		}
		parsed, _, err := parseFlagValue(target, flag.Value, schema)
		if err != nil {
			return LoadResult{}, fmt.Errorf("read flag %s: %w", flagSource(key, flag), err)
		}
		mergeLayer(result.values, nestedValue(target, parsed), Source{Kind: KindFlag, Origin: flagSource(key, flag)}, result.Sources)
	}
	if err := resolvePaths(result.values, result.Sources, workingDir, schema); err != nil {
		return LoadResult{}, err
	}
	result.Config, err = decodeConfig(result.values, Defaults())
	if err != nil {
		return LoadResult{}, fmt.Errorf("decode effective configuration: %w", err)
	}
	if application.Defaults != nil {
		result.Application, err = decodeSettings(result.values, application.Defaults)
		if err != nil {
			return LoadResult{}, fmt.Errorf("decode effective application settings: %w", err)
		}
		if application.Validate != nil {
			if err := application.Validate(result.Application); err != nil {
				return LoadResult{}, fmt.Errorf("validate application settings: %w", err)
			}
		}
	}
	return result, nil
}

func environmentKey(name string, schema map[string]valueType) string {
	for key := range schema {
		if name == "CLI_APP_"+strings.ToUpper(strings.ReplaceAll(key, ".", "_")) {
			return key
		}
	}
	return ""
}

func nestedValue(key string, value any) map[string]any {
	parts := strings.Split(key, ".")
	result := map[string]any{parts[len(parts)-1]: value}
	for i := len(parts) - 2; i >= 0; i-- {
		result = map[string]any{parts[i]: result}
	}
	return result
}

func parseOverlayValue(key, value string, schema map[string]valueType) (any, bool, error) {
	kind, known := schema[key]
	if !known {
		return nil, false, nil
	}
	switch kind {
	case stringValue, pathValue:
		return value, true, nil
	case durationValue:
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return nil, true, fmt.Errorf("invalid duration %q", value)
		}
		return parsed, true, nil
	case integerValue:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return nil, true, fmt.Errorf("invalid integer %q", value)
		}
		return parsed, true, nil
	case floatValue:
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, true, fmt.Errorf("invalid number %q", value)
		}
		return parsed, true, nil
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
	case durationValue:
		parsed, ok := value.(time.Duration)
		if !ok {
			return nil, true, fmt.Errorf("expected duration, got %T", value)
		}
		return parsed, true, nil
	case integerValue:
		parsed, ok := value.(int)
		if !ok {
			return nil, true, fmt.Errorf("expected integer, got %T", value)
		}
		return parsed, true, nil
	case floatValue:
		parsed, ok := value.(float64)
		if !ok {
			return nil, true, fmt.Errorf("expected number, got %T", value)
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
