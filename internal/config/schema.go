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
		"log.level":                       stringValue,
		"log.file.level":                  stringValue,
		"log.file.path":                   pathValue,
		"output":                          stringValue,
		"color":                           stringValue,
		"quiet":                           booleanValue,
		"timeouts.cleanup":                durationValue,
		"timeouts.demo.structured_output": durationValue,
		"timeouts.demo.not_found":         durationValue,
		"timeouts.demo.secret_credential": durationValue,
		"retry.demo.flaky_dependency.max_attempts":  integerValue,
		"retry.demo.flaky_dependency.initial_delay": durationValue,
		"retry.demo.flaky_dependency.max_delay":     durationValue,
		"retry.demo.flaky_dependency.jitter":        floatValue,
		"demo.secret_credential.reference":          stringValue,
	}
}
