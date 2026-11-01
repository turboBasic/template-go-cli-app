package logging

import "log/slog"

// Secret is a string value that is always redacted by structured logging.
type Secret string

// LogValue prevents the underlying secret from appearing in structured logs.
func (Secret) LogValue() slog.Value {
	return slog.StringValue("[REDACTED]")
}
