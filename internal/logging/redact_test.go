package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestSecretLogValue(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(redact(slog.NewJSONHandler(&output, nil)))
	logger.Info("connect", slog.Any("token", Secret("never-print-this")))
	if strings.Contains(output.String(), "never-print-this") || !strings.Contains(output.String(), "[REDACTED]") {
		t.Fatalf("log output = %q, want redacted secret", output.String())
	}
}

func TestRedactSensitiveAndNestedAttributes(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(redact(slog.NewJSONHandler(&output, nil)))
	logger.Info("connect",
		slog.String("host", "api.example.com"),
		slog.String("Access-Token", "token-secret"),
		slog.Group("credentials",
			slog.String("api_key", "key-secret"),
			slog.Group("database", slog.String("PASSWORD", "password-secret")),
		),
		slog.String("authorization", "auth-secret"),
	)
	got := output.String()
	for _, secret := range []string{"token-secret", "key-secret", "password-secret", "auth-secret"} {
		if strings.Contains(got, secret) {
			t.Errorf("log output leaks %q: %s", secret, got)
		}
	}
	for _, want := range []string{"api.example.com", "[REDACTED]"} {
		if !strings.Contains(got, want) {
			t.Errorf("log output = %s, want it to include %q", got, want)
		}
	}
}

func TestWithAttrsRedactsBoundAttributes(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(redact(slog.NewJSONHandler(&output, nil))).With(
		slog.String("access_token", "bound-secret"),
		slog.Group("connection", slog.String("authorization", "nested-bound-secret")),
	)
	logger.Info("connect", slog.String("host", "api.example.com"))
	if strings.Contains(output.String(), "bound-secret") || strings.Contains(output.String(), "nested-bound-secret") {
		t.Fatalf("log output leaks a bound secret: %s", output.String())
	}
	if !strings.Contains(output.String(), "api.example.com") {
		t.Fatalf("log output lost benign attribute: %s", output.String())
	}
}

func TestWithGroupRedactsAttributesInSensitiveGroup(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(redact(slog.NewJSONHandler(&output, nil))).WithGroup("tokens")
	logger.Info("connect", slog.String("primary", "group-secret"), slog.String("host", "api.example.com"))
	if strings.Contains(output.String(), "group-secret") || !strings.Contains(output.String(), "[REDACTED]") {
		t.Fatalf("log output = %q, want all values in sensitive group redacted", output.String())
	}
	if strings.Contains(output.String(), "api.example.com") {
		t.Fatalf("sensitive group leaked benign sibling: %s", output.String())
	}
}

func TestRedactingHandlerPreservesGroupContextWhenBound(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(redact(slog.NewJSONHandler(&output, nil))).WithGroup("request")
	logger = logger.With(slog.String("token", "group-bound-secret"))
	logger.InfoContext(context.Background(), "request")
	if strings.Contains(output.String(), "group-bound-secret") || !strings.Contains(output.String(), "[REDACTED]") {
		t.Fatalf("log output = %q, want grouped bound secret redacted", output.String())
	}
}
