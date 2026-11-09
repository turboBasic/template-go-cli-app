package config

import (
	"fmt"
	"reflect"
	"strings"
)

// ApplicationSettings registers a typed application's defaults and validation with the shared loader.
type ApplicationSettings struct {
	// Defaults is a struct whose koanf tags define the application's keys.
	Defaults any
	// Validate checks the decoded defaults and effective application settings.
	Validate func(any) error
}

func settingsSchema(defaults ...any) (map[string]valueType, error) {
	schema := make(map[string]valueType)
	for _, setting := range defaults {
		for key, kind := range schemaFor(setting) {
			for existing := range schema {
				if existing == key || strings.HasPrefix(existing, key+".") || strings.HasPrefix(key, existing+".") {
					return nil, fmt.Errorf("configuration schema collision between %q and %q", existing, key)
				}
			}
			schema[key] = kind
		}
	}
	return schema, nil
}

func schemaFor(settings any) map[string]valueType {
	typ := reflect.TypeOf(settings)
	if typ == nil || typ.Kind() != reflect.Struct {
		panic("configuration defaults must be a struct")
	}
	schema := make(map[string]valueType)
	var walk func(reflect.Type, string)
	walk = func(typ reflect.Type, prefix string) {
		for field := range typ.Fields() {
			key := prefix + field.Tag.Get("koanf")
			if field.Type.Kind() == reflect.Struct {
				walk(field.Type, key+".")
				continue
			}
			schema[key] = valueTypeOf(field)
		}
	}
	walk(typ, "")
	return schema
}

func mergedDefaults(application ApplicationSettings) map[string]any {
	shared := settingsMap(reflect.ValueOf(Defaults()))
	if application.Defaults == nil {
		return shared
	}
	mergeDefaults(shared, settingsMap(reflect.ValueOf(application.Defaults)))
	return shared
}

func mergeDefaults(shared, application map[string]any) {
	for key, value := range application {
		if nested, ok := value.(map[string]any); ok {
			current, exists := shared[key].(map[string]any)
			if !exists {
				current = make(map[string]any)
				shared[key] = current
			}
			mergeDefaults(current, nested)
			continue
		}
		shared[key] = value
	}
}
