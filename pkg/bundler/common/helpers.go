package common

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"
)

// TemplateRenderer provides template rendering functionality for bundlers.
type TemplateRenderer struct {
	// templateGetter is a function that retrieves template content by name.
	templateGetter func(name string) (string, bool)
}

// NewTemplateRenderer creates a new template renderer with the given template getter.
func NewTemplateRenderer(getter func(name string) (string, bool)) *TemplateRenderer {
	return &TemplateRenderer{
		templateGetter: getter,
	}
}

// Render renders a template with the given data.
func (r *TemplateRenderer) Render(name string, data map[string]interface{}) (string, error) {
	tmplContent, ok := r.templateGetter(name)
	if !ok {
		return "", fmt.Errorf("template %s not found", name)
	}

	tmpl, err := template.New(name).Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// FileWriter provides file writing functionality with result tracking.
type FileWriter struct {
	result *Result
}

// NewFileWriter creates a new file writer with the given result tracker.
func NewFileWriter(result *Result) *FileWriter {
	return &FileWriter{
		result: result,
	}
}

// WriteFile writes content to a file with the specified permissions and updates the result.
func (w *FileWriter) WriteFile(path string, content []byte, perm os.FileMode) error {
	if err := os.WriteFile(path, content, perm); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	w.result.AddFile(path, int64(len(content)))

	slog.Debug("file written",
		"path", path,
		"size_bytes", len(content),
		"permissions", perm,
	)

	return nil
}

// WriteFileString writes string content to a file with the specified permissions.
func (w *FileWriter) WriteFileString(path, content string, perm os.FileMode) error {
	return w.WriteFile(path, []byte(content), perm)
}

// MakeExecutable changes file permissions to make it executable.
func (w *FileWriter) MakeExecutable(path string) error {
	if err := os.Chmod(path, 0755); err != nil {
		w.result.AddError(fmt.Errorf("failed to make %s executable: %w", filepath.Base(path), err))
		return err
	}
	return nil
}

// DirectoryManager provides directory management functionality.
type DirectoryManager struct{}

// NewDirectoryManager creates a new directory manager.
func NewDirectoryManager() *DirectoryManager {
	return &DirectoryManager{}
}

// CreateDirectories creates multiple directories with the specified permissions.
func (m *DirectoryManager) CreateDirectories(dirs []string, perm os.FileMode) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, perm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// CreateBundleStructure creates the standard bundle directory structure.
// Returns the bundle root directory and subdirectories (scripts, manifests).
func (m *DirectoryManager) CreateBundleStructure(outputDir, bundleName string) (string, map[string]string, error) {
	bundleDir := filepath.Join(outputDir, bundleName)
	scriptsDir := filepath.Join(bundleDir, "scripts")
	manifestsDir := filepath.Join(bundleDir, "manifests")

	dirs := []string{bundleDir, scriptsDir, manifestsDir}
	if err := m.CreateDirectories(dirs, 0755); err != nil {
		return "", nil, err
	}

	subdirs := map[string]string{
		"root":      bundleDir,
		"scripts":   scriptsDir,
		"manifests": manifestsDir,
	}

	return bundleDir, subdirs, nil
}

// ContextChecker provides context cancellation checking.
type ContextChecker struct{}

// NewContextChecker creates a new context checker.
func NewContextChecker() *ContextChecker {
	return &ContextChecker{}
}

// Check checks if the context has been cancelled.
func (c *ContextChecker) Check(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// ComputeChecksum computes the SHA256 checksum of the given content.
func ComputeChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// ChecksumGenerator generates checksums for bundle files.
type ChecksumGenerator struct {
	result *Result
}

// NewChecksumGenerator creates a new checksum generator.
func NewChecksumGenerator(result *Result) *ChecksumGenerator {
	return &ChecksumGenerator{
		result: result,
	}
}

// Generate generates a checksums file for all files in the result.
func (g *ChecksumGenerator) Generate(outputDir, title string) (string, error) {
	var content bytes.Buffer
	content.WriteString(fmt.Sprintf("# %s Bundle Checksums (SHA256)\n", title))
	content.WriteString(fmt.Sprintf("# Generated: %s\n\n", g.result.GeneratedAt()))

	for _, file := range g.result.Files {
		// Skip checksums file itself
		if filepath.Base(file) == "checksums.txt" {
			continue
		}

		fileContent, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s for checksum: %w", file, err)
		}

		checksum := ComputeChecksum(fileContent)

		// Get relative path from output directory
		relPath, err := filepath.Rel(outputDir, file)
		if err != nil {
			relPath = filepath.Base(file)
		}

		content.WriteString(fmt.Sprintf("%s  %s\n", checksum, relPath))
	}

	return content.String(), nil
}
