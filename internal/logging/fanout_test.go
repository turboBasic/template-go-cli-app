package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestFanoutSendsSanitizedRecordsToEveryHandler(t *testing.T) {
	var terminal, file bytes.Buffer
	logger := slog.New(redact(fanout(
		slog.NewJSONHandler(&terminal, nil),
		slog.NewJSONHandler(&file, nil),
	)))
	logger.WithGroup("request").With(slog.String("password", "bound-secret")).Info(
		"handled",
		slog.String("token", "record-secret"),
		slog.String("host", "api.example.com"),
	)

	for name, output := range map[string]string{"terminal": terminal.String(), "file": file.String()} {
		for _, secret := range []string{"bound-secret", "record-secret"} {
			if strings.Contains(output, secret) {
				t.Errorf("%s output leaks %q: %s", name, secret, output)
			}
		}
		for _, want := range []string{"[REDACTED]", "api.example.com"} {
			if !strings.Contains(output, want) {
				t.Errorf("%s output = %s, want %q", name, output, want)
			}
		}
	}
}
