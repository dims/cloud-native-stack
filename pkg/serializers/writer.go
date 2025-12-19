package serializers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Format represents the output format type
type Format string

const (
	// FormatJSON outputs data in JSON format
	FormatJSON Format = "json"
	// FormatYAML outputs data in YAML format
	FormatYAML Format = "yaml"
	// FormatTable outputs data in table format
	FormatTable Format = "table"
)

// Writer handles serialization of configuration data to various formats.
type Writer struct {
	format Format
	output io.Writer
}

// NewWriter creates a new Writer with the specified format and output destination.
// If output is nil, os.Stdout will be used.
func NewWriter(format Format, output io.Writer) *Writer {
	if output == nil {
		output = os.Stdout
	}
	return &Writer{
		format: format,
		output: output,
	}
}

// Serialize outputs the given configuration data in the configured format.
func (w *Writer) Serialize(config any) error {
	switch w.format {
	case FormatJSON:
		return w.serializeJSON(config)
	case FormatYAML:
		return w.serializeYAML(config)
	case FormatTable:
		return w.serializeTable(config)
	default:
		return fmt.Errorf("unsupported format: %s", w.format)
	}
}

func (w *Writer) serializeJSON(config any) error {
	encoder := json.NewEncoder(w.output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to serialize to JSON: %w", err)
	}
	return nil
}

func (w *Writer) serializeYAML(config any) error {
	encoder := yaml.NewEncoder(w.output)
	encoder.SetIndent(2)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to serialize to YAML: %w", err)
	}
	return nil
}

func (w *Writer) serializeTable(config any) error {
	// Simple table implementation
	fmt.Fprintln(w.output, "Configuration Snapshot:")
	fmt.Fprintln(w.output, "----------------------")

	// Type assertion to our expected structure
	configs, ok := config.([]interface{})
	if !ok {
		return fmt.Errorf("unsupported config type for table format")
	}

	for i, cfg := range configs {
		fmt.Fprintf(w.output, "\n[%d] %+v\n", i+1, cfg)
	}

	return nil
}
