package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/NVIDIA/cloud-native-stack/pkg/logging"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/server"
)

const (
	name           = "eidos-api-server"
	versionDefault = "dev"
)

var (
	// overridden during build with ldflags to reflect actual version info
	// e.g., -X "github.com/NVIDIA/cloud-native-stack/pkg/api.version=1.0.0"
	version = versionDefault
	commit  = "unknown"
	date    = "unknown"
)

// Serve starts the API server and blocks until shutdown.
// It configures logging, sets up routes, and handles graceful shutdown.
// Returns an error if the server fails to start or encounters a fatal error.
func Serve() error {
	ctx := context.Background()

	logging.SetDefaultStructuredLogger(name, version)
	slog.Info("starting",
		"name", name,
		"version", version,
		"commit", commit,
		"date", date,
	)

	// Setup recipe handler
	b := recipe.NewBuilder()

	r := map[string]http.HandlerFunc{
		"/v1/recipe": b.HandleRecipes,
	}

	// Create and run server
	s := server.New(
		server.WithName(name),
		server.WithVersion(version),
		server.WithHandler(r),
	)

	if err := s.Run(ctx); err != nil {
		slog.Error("server exited with error", "error", err)
		return err
	}

	return nil
}
