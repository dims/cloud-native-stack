package logging

import (
	"log/slog"
	"os"
)

// New creates a new structured logger with the specified level.
// It uses JSON format for machine-readable logs written to stderr.
func New(level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, opts))
}

// ParseLevel converts a string to a slog.Level.
// Returns slog.LevelInfo if the level is not recognized.
func ParseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
