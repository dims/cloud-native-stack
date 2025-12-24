package server

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/time/rate"
)

// Config holds server configuration
type Config struct {
	// Server identity
	Name string

	// Additional Handlers to be added to the server
	Handlers map[string]http.HandlerFunc

	// Server configuration
	Address string
	Port    int

	// Rate limiting configuration
	RateLimit      rate.Limit // requests per second
	RateLimitBurst int        // burst size

	// Request limits
	MaxBulkRequests int

	// Timeouts
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// NewConfig returns a new Config with sensible defaults.
// Use this when you want to customize config programmatically.
func NewConfig() *Config {
	return parseConfig()
}

// parseConfig returns sensible defaults
func parseConfig() *Config {
	cfg := &Config{
		Name:            "server",
		Address:         "",
		Port:            8080,
		RateLimit:       100, // 100 req/s
		RateLimitBurst:  200, // burst of 200
		MaxBulkRequests: 100,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}

	// Override with environment variables if set
	if portStr := os.Getenv("PORT"); portStr != "" {
		var port int
		if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil {
			cfg.Port = port
		}
	}

	return cfg
}
