package config

import (
	"fmt"
	"reflect"
	"time"
)

type valueType uint8

const (
	stringValue valueType = iota
	booleanValue
	pathValue
	durationValue
	integerValue
	floatValue
)

// productionSchema maps every dotted setting key to its value type. It is derived from the koanf tags
// and field types of Config, so a new field is a new setting; a string field tagged config:"path" is a path.
func productionSchema() map[string]valueType {
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
	walk(reflect.TypeFor[Config](), "")
	return schema
}

func valueTypeOf(field reflect.StructField) valueType {
	typ := field.Type
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch {
	case typ == reflect.TypeFor[time.Duration]():
		return durationValue
	case typ.Kind() == reflect.String && field.Tag.Get("config") == "path":
		return pathValue
	case typ.Kind() == reflect.String:
		return stringValue
	case typ.Kind() == reflect.Bool:
		return booleanValue
	case typ.Kind() == reflect.Int:
		return integerValue
	case typ.Kind() == reflect.Float64:
		return floatValue
	}
	// A programmer error caught by any test that loads configuration, not an operational failure.
	panic(fmt.Sprintf("config field %s has unsupported type %s", field.Name, field.Type))
}

// defaultValues is Defaults() as nested maps. A nil optional setting stays nil so inspection lists
// every supported key and renders it as unset.
func defaultValues() map[string]any {
	return settingsMap(reflect.ValueOf(Defaults()))
}

func settingsMap(settings reflect.Value) map[string]any {
	group := make(map[string]any, settings.NumField())
	for i := range settings.NumField() {
		key := settings.Type().Field(i).Tag.Get("koanf")
		value := settings.Field(i)
		switch {
		case value.Kind() == reflect.Struct:
			group[key] = settingsMap(value)
		case value.Kind() == reflect.Pointer && value.IsNil():
			group[key] = nil
		case value.Kind() == reflect.Pointer:
			group[key] = value.Elem().Interface()
		default:
			group[key] = value.Interface()
		}
	}
	return group
}
