package config

type valueType uint8

const (
	stringValue valueType = iota
	booleanValue
	pathValue
)

func productionSchema() map[string]valueType {
	return map[string]valueType{
		"log-level": stringValue,
		"output":    stringValue,
		"color":     stringValue,
		"quiet":     booleanValue,
	}
}
