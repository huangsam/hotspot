// Package logger provides a thin wrapper around log/slog for centralized
// telemetry and diagnostic logging in the Hotspot CLI.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Level is the dynamic logging level for the global logger.
var Level = new(slog.LevelVar)

// isQuiet suppresses all non-error output when true.
var isQuiet bool

// InitLogger with a default level on package load.
func init() {
	InitLogger("warn")
}

// SetQuiet enables or disables quiet mode.
func SetQuiet(q bool) {
	isQuiet = q
}

// IsQuiet returns whether quiet mode is enabled.
func IsQuiet() bool {
	return isQuiet
}

// InitLogger initializes the global default logger to write to os.Stderr
// according to the provided log level. Accepted values: warn, info, debug
// (case-insensitive). Any unrecognized value defaults to warn.
func InitLogger(level string) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		Level.Set(slog.LevelDebug)
	case "info":
		Level.Set(slog.LevelInfo)
	case "warn":
		Level.Set(slog.LevelWarn)
	case "error":
		Level.Set(slog.LevelError)
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

// Info logs an informational message to stderr.
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Debug logs a debug message to stderr.
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Error logs an error message to stderr.
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
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
