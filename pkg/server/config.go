package server

import (
	"fmt"
	"log/slog"
	"os"
	"time"
)

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	cfg := &Config{
		Address:         "",
		Port:            8080,
		RateLimit:       100, // 100 req/s
		RateLimitBurst:  200, // burst of 200
		CacheMaxAge:     300, // 5 minutes
		MaxBulkRequests: 100,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		LogLevel:        slog.LevelInfo.String(),
	}

	// Override with environment variables if set
	if portStr := os.Getenv("PORT"); portStr != "" {
		var port int
		if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil {
			cfg.Port = port
		}
	}

	if logLevelStr := os.Getenv("LOG_LEVEL"); logLevelStr != "" {
		cfg.LogLevel = logLevelStr
	}

	return cfg
}
