package logging_test

import (
	"log/slog"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/logging"
)

func TestNew(t *testing.T) {
	logger := logging.New(slog.LevelInfo)
	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"invalid", slog.LevelInfo}, // defaults to info
		{"", slog.LevelInfo},        // defaults to info
		{"DEBUG", slog.LevelInfo},   // case sensitive, defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := logging.ParseLevel(tt.input)
			if level != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, level, tt.expected)
			}
		})
	}
}

func TestNew_DifferentLevels(t *testing.T) {
	levels := []slog.Level{
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
	}

	for _, level := range levels {
		logger := logging.New(level)
		if logger == nil {
			t.Errorf("Expected non-nil logger for level %v", level)
		}
	}
}
