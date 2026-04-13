// Package logger provides a thin wrapper around log/slog for centralized
// telemetry and diagnostic logging in the Hotspot CLI.
package logger

import (
	"log/slog"
	"os"
)

// Level is the dynamic logging level for the global logger.
var Level = new(slog.LevelVar)

// InitLogger initializes the global default logger to write to os.Stderr
// according to the provided verbosity settings.
func InitLogger(verbose, debug bool) {
	switch {
	case debug:
		Level.Set(slog.LevelDebug)
	case verbose:
		Level.Set(slog.LevelInfo)
	default:
		Level.Set(slog.LevelWarn)
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: Level,
	})
	slog.SetDefault(slog.New(handler))
}

// Warn logs a warning message to stderr.
func Warn(msg string, err error) {
	if err != nil {
		slog.Warn(msg, "error", err)
	} else {
		slog.Warn(msg)
	}
}

// Fatal logs a fatal error message to stderr and exits the program.
func Fatal(msg string, err error) {
	if err != nil {
		slog.Error(msg, "error", err)
	} else {
		slog.Error(msg)
	}
	os.Exit(1)
}
