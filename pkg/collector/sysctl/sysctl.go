package sysctl

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

var (
	// Keys to filter out from systemd properties for privacy/security or noise reduction
	filterOutSysctlKeys = []string{
		"/proc/sys/dev/cdrom/*",
	}
)

// Collector collects sysctl configurations from /proc/sys
// excluding /proc/sys/net
type Collector struct {
}

// Collect gathers sysctl configurations from /proc/sys, excluding /proc/sys/net
// and returns them as a single Configuration with a map of all parameters.
func (s *Collector) Collect(ctx context.Context) (*measurement.Measurement, error) {
	root := "/proc/sys"
	params := make(map[string]measurement.Reading)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk dir: %w", err)
		}

		// Check if context is canceled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Skip symlinks to prevent directory traversal attacks
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Ensure path is under root (defense in depth)
		if !strings.HasPrefix(path, root) {
			return fmt.Errorf("path traversal detected: %s", path)
		}

		if strings.HasPrefix(path, "/proc/sys/net") {
			return nil
		}

		c, err := os.ReadFile(path)
		if err != nil {
			// Skip files we can't read (some proc files are write-only or restricted)
			return nil
		}

		content := strings.TrimSpace(string(c))

		// Check if content has multiple lines with space-separated values
		lines := strings.Split(content, "\n")
		if len(lines) > 1 {
			// Try to parse as key-value pairs
			allParsed := true
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				// Check if line has space-separated key and value
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					// Create new entry with extended path
					key := parts[0]
					value := strings.Join(parts[1:], " ")
					extendedPath := path + "/" + key
					params[extendedPath] = measurement.Str(value)
				} else {
					// Not a key-value pair format
					allParsed = false
					break
				}
			}

			// If all lines were parsed, skip the original entry
			if allParsed {
				return nil
			}
		}

		// Store original content if not multi-line key-value format
		params[path] = measurement.Str(content)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to capture sysctl config: %w", err)
	}

	res := &measurement.Measurement{
		Type: measurement.TypeSysctl,
		Subtypes: []measurement.Subtype{
			{
				Data: measurement.FilterOut(params, filterOutSysctlKeys),
			},
		},
	}

	return res, nil
}
