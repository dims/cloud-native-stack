package os

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

// collectKMod retrieves the list of loaded kernel modules from /proc/modules
// and returns them as a subtype with module names as keys.
func (c *Collector) collectKMod(ctx context.Context) (*measurement.Subtype, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root := "/proc/modules"

	content, err := os.ReadFile(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read kernel modules: %w", err)
	}

	readings := make(map[string]measurement.Reading)
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Module name is the first field
		fields := strings.Fields(line)
		if len(fields) > 0 {
			readings[fields[0]] = measurement.Bool(true)
		}
	}

	res := &measurement.Subtype{
		Name: "kmod",
		Data: readings,
	}

	return res, nil
}
