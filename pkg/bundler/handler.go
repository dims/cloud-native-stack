package bundler

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	cnserrors "github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/server"
)

const (
	// DefaultBundleTimeout is the timeout for bundle generation.
	// Bundle generation involves parallel file I/O and template rendering.
	DefaultBundleTimeout = 60 * time.Second
)

// HandleBundles processes bundle generation requests.
// It accepts a POST request with a JSON body containing the recipe (RecipeResult).
// Bundlers can be specified via the "bundlers" query parameter (comma-delimited).
// If no bundlers are specified, all registered bundlers are executed.
// The response is a zip archive containing all generated bundles.
//
// Example:
//
//	POST /v1/bundle?bundlers=gpu-operator,network-operator
//	Content-Type: application/json
//	Body: { "apiVersion": "cns.nvidia.com/v1alpha1", "kind": "Recipe", ... }
func (b *DefaultBundler) HandleBundles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		server.WriteError(w, r, http.StatusMethodNotAllowed, cnserrors.ErrCodeMethodNotAllowed,
			"Method not allowed", false, map[string]interface{}{
				"method": r.Method,
			})
		return
	}

	// Add request-scoped timeout
	ctx, cancel := context.WithTimeout(r.Context(), DefaultBundleTimeout)
	defer cancel()

	// Parse bundlers from query parameter (comma-delimited)
	bundlersParam := r.URL.Query().Get("bundlers")
	var bundlerNames []string
	if bundlersParam != "" {
		bundlerNames = strings.Split(bundlersParam, ",")
	}

	// Parse request body directly as RecipeResult
	var recipeResult recipe.RecipeResult
	if err := json.NewDecoder(r.Body).Decode(&recipeResult); err != nil {
		server.WriteError(w, r, http.StatusBadRequest, cnserrors.ErrCodeInvalidRequest,
			"Invalid request body", false, map[string]interface{}{
				"error": err.Error(),
			})
		return
	}

	// Validate recipe has component references
	if len(recipeResult.ComponentRefs) == 0 {
		server.WriteError(w, r, http.StatusBadRequest, cnserrors.ErrCodeInvalidRequest,
			"Recipe must contain at least one component reference", false, nil)
		return
	}

	slog.Debug("bundle request received",
		"components", len(recipeResult.ComponentRefs),
		"bundlers", bundlerNames,
	)

	// Parse bundler types
	bundlerTypes := make([]types.BundleType, 0, len(bundlerNames))
	for _, bt := range bundlerNames {
		bt = strings.TrimSpace(bt)
		if bt == "" {
			continue
		}
		parsed, err := types.ParseType(bt)
		if err != nil {
			server.WriteError(w, r, http.StatusBadRequest, cnserrors.ErrCodeInvalidRequest,
				"Invalid bundler type", false, map[string]interface{}{
					"bundler": bt,
					"error":   err.Error(),
					"valid":   types.SupportedBundleTypesAsStrings(),
				})
			return
		}
		bundlerTypes = append(bundlerTypes, parsed)
	}

	// Create temporary directory for bundle output
	tempDir, err := os.MkdirTemp("", "eidos-bundle-*")
	if err != nil {
		server.WriteError(w, r, http.StatusInternalServerError, cnserrors.ErrCodeInternal,
			"Failed to create temporary directory", true, nil)
		return
	}
	defer os.RemoveAll(tempDir) // Clean up on exit

	// Create a new bundler with specified types (or use all if empty)
	bundler := New(
		WithBundlerTypes(bundlerTypes),
		WithFailFast(false), // Collect all errors
	)

	// Generate bundles
	output, err := bundler.Make(ctx, &recipeResult, tempDir)
	if err != nil {
		server.WriteErrorFromErr(w, r, err, "Failed to generate bundles", nil)
		return
	}

	// Check for bundle errors
	if output.HasErrors() {
		errorDetails := make([]map[string]interface{}, 0, len(output.Errors))
		for _, be := range output.Errors {
			errorDetails = append(errorDetails, map[string]interface{}{
				"bundler": be.BundlerType,
				"error":   be.Error,
			})
		}
		server.WriteError(w, r, http.StatusInternalServerError, cnserrors.ErrCodeInternal,
			"Some bundlers failed", true, map[string]interface{}{
				"errors":        errorDetails,
				"success_count": output.SuccessCount(),
			})
		return
	}

	// Stream zip response
	if err := streamZipResponse(w, tempDir, output); err != nil {
		// Can't write error response if we've already started writing
		slog.Error("failed to stream zip response", "error", err)
		return
	}
}

// streamZipResponse creates a zip archive from the output directory and streams it to the response.
func streamZipResponse(w http.ResponseWriter, dir string, output *result.Output) error {
	// Set response headers before writing body
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"bundles.zip\"")
	w.Header().Set("X-Bundle-Files", strconv.Itoa(output.TotalFiles))
	w.Header().Set("X-Bundle-Size", strconv.FormatInt(output.TotalSize, 10))
	w.Header().Set("X-Bundle-Duration", output.TotalDuration.String())

	// Create zip writer directly to response
	zw := zip.NewWriter(w)
	defer zw.Close()

	// Walk the directory and add all files to zip
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk error: %w", err)
		}

		// Skip the root directory itself
		if path == dir {
			return nil
		}

		// Get relative path for zip entry
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Create zip file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to create file header: %w", err)
		}
		header.Name = relPath

		// Preserve directory structure
		if info.IsDir() {
			header.Name += "/"
			_, headerErr := zw.CreateHeader(header)
			return headerErr
		}

		// Use deflate compression
		header.Method = zip.Deflate

		writer, err := zw.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to create zip entry: %w", err)
		}

		// Open and copy file content
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		if err != nil {
			return fmt.Errorf("failed to copy file content: %w", err)
		}

		return nil
	})
}
