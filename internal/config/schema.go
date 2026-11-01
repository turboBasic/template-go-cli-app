package config

type valueType uint8

const (
	stringValue valueType = iota
	booleanValue
	pathValue
	durationValue
	integerValue
	floatValue
)

func productionSchema() map[string]valueType {
	return map[string]valueType{
		"log.console.level":                   stringValue,
		"log.file.level":                      stringValue,
		"log.file.path":                       pathValue,
		"output":                              stringValue,
		"color":                               stringValue,
		"quiet":                               booleanValue,
		"timeouts.cleanup":                    durationValue,
		"timeouts.resource.list":              durationValue,
		"timeouts.resource.status":            durationValue,
		"timeouts.resource.apply":             durationValue,
		"retry.resource.status.max_attempts":  integerValue,
		"retry.resource.status.initial_delay": durationValue,
		"retry.resource.status.max_delay":     durationValue,
		"retry.resource.status.jitter":        floatValue,
	}
}
