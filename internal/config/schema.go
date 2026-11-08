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
		"log.level":                          stringValue,
		"log.file.level":                     stringValue,
		"log.file.path":                      pathValue,
		"output":                             stringValue,
		"color":                              stringValue,
		"quiet":                              booleanValue,
		"timeouts.cleanup":                   durationValue,
		"timeouts.demo.structured_output":    durationValue,
		"timeouts.demo.not_found":            durationValue,
		"timeouts.demo.secret_credential":    durationValue,
		"retry.demo.not_found.max_attempts":  integerValue,
		"retry.demo.not_found.initial_delay": durationValue,
		"retry.demo.not_found.max_delay":     durationValue,
		"retry.demo.not_found.jitter":        floatValue,
		"demo.secret_credential.reference":   stringValue,
	}
}
