package collectors

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SysctlCollector collects sysctl configurations from /proc/sys
// excluding /proc/sys/net
type SysctlCollector struct {
}

// SysctlType is the type identifier for sysctl configurations
const SysctlType string = "Sysctl"

// SysctlConfig represents a single sysctl configuration entry
// with its key and value
type SysctlConfig struct {
	Key   string
	Value string
}

// Collect gathers sysctl configurations from /proc/sys, excluding /proc/sys/net
// and returns them as a slice of Configuration objects.
func (s *SysctlCollector) Collect(ctx context.Context) ([]Configuration, error) {
	root := "/proc/sys"
	res := make([]Configuration, 0, 500)

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

		res = append(res, Configuration{
			Type: SysctlType,
			Data: SysctlConfig{
				Key:   path,
				Value: strings.TrimSpace(string(c)),
			},
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to capture sysctl config: %w", err)
	}

	return res, nil
}
