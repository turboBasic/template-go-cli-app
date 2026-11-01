package logging

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/config"
)

func TestDefaultLoggingDestinations(t *testing.T) {
	var stderr bytes.Buffer
	result, err := newWithWriters(config.Defaults().Log, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	result.Logger.Debug("file-only debug")
	result.Logger.Warn("warning")
	if err := result.Close(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stderr.String(), "file-only debug") || !strings.Contains(stderr.String(), "warning") {
		t.Fatalf("stderr output = %q, want WARN only", stderr.String())
	}
	if strings.HasPrefix(strings.TrimSpace(stderr.String()), "{") {
		t.Fatalf("stderr should be human-readable text, got %q", stderr.String())
	}
}

func TestFileLoggingIsJSONLinesWithIndependentLevelsAndRestrictedMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.jsonl")
	cfg := config.Defaults().Log
	cfg.File.Path = path
	var stderr bytes.Buffer
	result, err := newWithWriters(cfg, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	result.Logger.Debug("file debug", "token", "file-secret")
	result.Logger.Warn("shared warning")
	if err := result.Close(); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(stderr.String(), "file debug") || !strings.Contains(stderr.String(), "shared warning") {
		t.Fatalf("stderr filtering/output = %q, want WARN and no DEBUG", stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "file-secret") {
		t.Fatalf("file output leaked secret: %s", data)
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var lines int
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("JSONL line %q: %v", scanner.Text(), err)
		}
		lines++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if lines != 2 || !strings.Contains(string(data), "file debug") || !strings.Contains(string(data), "[REDACTED]") {
		t.Fatalf("file lines/output = %d/%s, want both DEBUG and WARN with redaction", lines, data)
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm() & 0o077; got != 0 {
			t.Fatalf("file permissions = %04o, want no group/other bits", info.Mode().Perm())
		}
	}
}

func TestFileLoggingInitializationFailures(t *testing.T) {
	tests := []struct {
		name string
		edit func(*config.LogConfig, string)
	}{
		{name: "invalid level", edit: func(cfg *config.LogConfig, path string) { cfg.File.Path, cfg.File.Level = path, "trace" }},
		{name: "missing parent directory", edit: func(cfg *config.LogConfig, path string) { cfg.File.Path = filepath.Join(path, "missing", "app.jsonl") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := config.Defaults().Log
			path := filepath.Join(t.TempDir(), "app.jsonl")
			test.edit(&cfg, path)
			if _, err := newWithWriters(cfg, &bytes.Buffer{}); err == nil {
				t.Fatal("newWithWriters() error = nil, want initialization error")
			}
		})
	}
}

func TestCloseWithoutFileIsSafe(t *testing.T) {
	result, err := newWithWriters(config.Defaults().Log, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := result.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
}
