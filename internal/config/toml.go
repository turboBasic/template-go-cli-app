package config

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/v2"
)

// DecodeTOML decodes and validates one configuration file before it is merged.
func DecodeTOML(source string, data []byte) (Config, error) {
	schema, err := schemaFor(Defaults())
	if err != nil {
		return Config{}, err
	}
	values, err := parseTOML(source, data, schema)
	if err != nil {
		return Config{}, err
	}
	config, err := decodeConfig(values, Defaults())
	if err != nil {
		return Config{}, fmt.Errorf("decode configuration %s: %w", source, err)
	}
	return config, nil
}

// DecodeTOMLWithSettings decodes shared config and one registered application's typed settings.
func DecodeTOMLWithSettings(source string, data []byte, application ApplicationSettings) (Config, any, error) {
	settings := []any{Defaults()}
	if application.Defaults != nil {
		settings = append(settings, application.Defaults)
	}
	schema, err := settingsSchema(settings...)
	if err != nil {
		return Config{}, nil, err
	}
	values, err := parseTOML(source, data, schema)
	if err != nil {
		return Config{}, nil, err
	}
	shared, err := decodeConfig(values, Defaults())
	if err != nil {
		return Config{}, nil, fmt.Errorf("decode configuration %s: %w", source, err)
	}
	var appSettings any
	if application.Defaults != nil {
		appSettings, err = decodeSettings(values, application.Defaults)
		if err != nil {
			return Config{}, nil, fmt.Errorf("decode application configuration %s: %w", source, err)
		}
		if application.Validate != nil {
			if err := application.Validate(appSettings); err != nil {
				return Config{}, nil, fmt.Errorf("validate application configuration %s: %w", source, err)
			}
		}
	}
	return shared, appSettings, nil
}

func parseTOML(source string, data []byte, schema map[string]valueType) (map[string]any, error) {
	values, err := toml.Parser().Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("decode configuration %s: %w", source, err)
	}
	if err := validateValues(source, values, schema, ""); err != nil {
		return nil, err
	}
	return values, nil
}

func validateValues(source string, values map[string]any, schema map[string]valueType, prefix string) error {
	for key, value := range values {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		if nested, ok := value.(map[string]any); ok {
			if _, scalar := schema[path]; scalar {
				return fmt.Errorf("decode configuration %s: invalid type for key %q", source, path)
			}
			if !knownTable(schema, path) {
				return fmt.Errorf("decode configuration %s: unknown key %q", source, path)
			}
			if err := validateValues(source, nested, schema, path); err != nil {
				return err
			}
			continue
		}
		kind, known := schema[path]
		if !known {
			return fmt.Errorf("decode configuration %s: unknown key %q", source, path)
		}
		var valid bool
		switch kind {
		case stringValue, pathValue, durationValue:
			_, valid = value.(string)
		case integerValue:
			_, valid = value.(int64)
		case floatValue:
			_, valid = value.(float64)
		case booleanValue:
			_, valid = value.(bool)
		default:
			return fmt.Errorf("decode configuration %s: unsupported type for key %q", source, path)
		}
		if !valid {
			return fmt.Errorf("decode configuration %s: invalid type for key %q", source, path)
		}
	}
	return nil
}

// knownTable reports whether path is a prefix of a schema key, so an empty table is checked like a leaf.
func knownTable(schema map[string]valueType, path string) bool {
	for key := range schema {
		if strings.HasPrefix(key, path+".") {
			return true
		}
	}
	return false
}

func decodeConfig(values map[string]any, defaults Config) (Config, error) {
	schema, err := schemaFor(defaults)
	if err != nil {
		return Config{}, err
	}
	for key, kind := range schema {
		if kind != durationValue {
			continue
		}
		if err := decodeDuration(values, key); err != nil {
			return Config{}, err
		}
	}
	settings := koanf.New(".")
	for key, value := range values {
		if err := settings.Set(key, value); err != nil {
			return Config{}, fmt.Errorf("decode key %q: %w", key, err)
		}
	}
	config := defaults
	if err := settings.Unmarshal("", &config); err != nil {
		return Config{}, err
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func decodeSettings(values map[string]any, defaults any) (any, error) {
	schema, err := schemaFor(defaults)
	if err != nil {
		return nil, err
	}
	for key, kind := range schema {
		if kind == durationValue {
			if err := decodeDuration(values, key); err != nil {
				return nil, err
			}
		}
	}
	settings := koanf.New(".")
	for key, value := range values {
		if err := settings.Set(key, value); err != nil {
			return nil, fmt.Errorf("decode key %q: %w", key, err)
		}
	}
	value := reflect.New(reflect.TypeOf(defaults))
	value.Elem().Set(reflect.ValueOf(defaults))
	if err := settings.Unmarshal("", value.Interface()); err != nil {
		return nil, err
	}
	return value.Elem().Interface(), nil
}

func decodeDuration(values map[string]any, key string) error {
	parts := strings.Split(key, ".")
	current := values
	for _, part := range parts[:len(parts)-1] {
		nested, ok := current[part].(map[string]any)
		if !ok {
			return nil
		}
		current = nested
	}
	value, ok := current[parts[len(parts)-1]]
	if !ok {
		return nil
	}
	if value == nil {
		return nil
	}
	if _, ok := value.(time.Duration); ok {
		return nil
	}
	text, ok := value.(string)
	if !ok {
		return fmt.Errorf("invalid type for key %q: expected duration string", key)
	}
	duration, err := time.ParseDuration(text)
	if err != nil {
		return fmt.Errorf("invalid duration for key %q: %w", key, err)
	}
	current[parts[len(parts)-1]] = duration
	return nil
}
