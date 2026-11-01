package logging

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/turboBasic/template-go-cli-app/internal/config"
)

// Result contains the configured logger and cleanup for owned resources.
type Result struct {
	Logger *slog.Logger
	Close  func() error
}

// New constructs the stderr logger and optional JSON Lines file destination.
func New(cfg config.LogConfig) (Result, error) {
	return newWithWriters(cfg, os.Stderr)
}

func newWithWriters(cfg config.LogConfig, stderr io.Writer) (Result, error) {
	consoleLevel, err := parseLevel(cfg.Console.Level)
	if err != nil {
		return Result{}, fmt.Errorf("configure console logger: %w", err)
	}
	handlers := []slog.Handler{
		slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: consoleLevel}),
	}
	var file *os.File
	if cfg.File.Path != "" {
		fileLevel, err := parseLevel(cfg.File.Level)
		if err != nil {
			return Result{}, fmt.Errorf("configure file logger: %w", err)
		}
		file, err = os.OpenFile(cfg.File.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return Result{}, fmt.Errorf("open log file %q: %w", cfg.File.Path, err)
		}
		if err := file.Chmod(0o600); err != nil {
			closeErr := file.Close()
			return Result{}, fmt.Errorf("set log file permissions %q: %w", cfg.File.Path, errors.Join(err, closeErr))
		}
		handlers = append(handlers, slog.NewJSONHandler(file, &slog.HandlerOptions{Level: fileLevel}))
	}
	result := Result{
		Logger: slog.New(redact(fanout(handlers...))),
		Close: func() error {
			if file == nil {
				return nil
			}
			return file.Close()
		},
	}
	return result, nil
}

func parseLevel(level string) (slog.Level, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level %q", level)
	}
}
