package config

import (
	"fmt"

	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/v2"
)

// DecodeTOML decodes and validates one configuration file before it is merged.
func DecodeTOML(source string, data []byte) (Config, error) {
	values, err := decodeTOMLWithSchema(source, data, productionSchema())
	if err != nil {
		return Config{}, err
	}
	config, err := decodeConfig(values)
	if err != nil {
		return Config{}, fmt.Errorf("decode configuration %s: %w", source, err)
	}
	return config, nil
}

func decodeTOMLWithSchema(source string, data []byte, schema map[string]valueType) (map[string]any, error) {
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
		case stringValue, pathValue:
			_, valid = value.(string)
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

func decodeConfig(values map[string]any) (Config, error) {
	settings := koanf.New(".")
	for key, value := range values {
		if err := settings.Set(key, value); err != nil {
			return Config{}, fmt.Errorf("decode key %q: %w", key, err)
		}
	}
	config := Defaults()
	if err := settings.Unmarshal("", &config); err != nil {
		return Config{}, err
	}
	config.LogLevel = config.Log.Console.Level
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}
