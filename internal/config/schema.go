package config

type valueType uint8

const (
	stringValue valueType = iota
	booleanValue
	pathValue
)

func productionSchema() map[string]valueType {
	return map[string]valueType{
		"log.console.level": stringValue,
		"log.file.level":    stringValue,
		"log.file.path":     pathValue,
		"output":            stringValue,
		"color":             stringValue,
		"quiet":             booleanValue,
	}
}
