package config

import (
	"fmt"

	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/v2"
)

// DecodeTOML decodes and validates one configuration file before it is merged.
func DecodeTOML(source string, data []byte) (Config, error) {
	values, err := toml.Parser().Unmarshal(data)
	if err != nil {
		return Config{}, fmt.Errorf("decode configuration %s: %w", source, err)
	}
	for key, value := range values {
		var valid bool
		switch key {
		case "log-level", "output", "color":
			_, valid = value.(string)
		case "quiet":
			_, valid = value.(bool)
		default:
			return Config{}, fmt.Errorf("decode configuration %s: unknown key %q", source, key)
		}
		if !valid {
			return Config{}, fmt.Errorf("decode configuration %s: invalid type for key %q", source, key)
		}
	}

	settings := koanf.New(".")
	for key, value := range values {
		if err := settings.Set(key, value); err != nil {
			return Config{}, fmt.Errorf("decode configuration %s key %q: %w", source, key, err)
		}
	}
	config := Defaults()
	if err := settings.Unmarshal("", &config); err != nil {
		return Config{}, fmt.Errorf("decode configuration %s: %w", source, err)
	}
	if err := config.Validate(); err != nil {
		return Config{}, fmt.Errorf("decode configuration %s: %w", source, err)
	}
	return config, nil
}
