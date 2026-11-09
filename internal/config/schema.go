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

// valueTypeOf derives a setting's supported value type from its struct field.
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
