package collectors

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// KModCollector collects information about loaded kernel modules from /proc/modules
// and parses them into KModConfig structures
type KModCollector struct {
}

// KModType is the type identifier for kernel module configurations
const KModType string = "KMod"

// KModConfig represents the configuration of a loaded kernel module
// with its name
type KModConfig struct {
	Name string
}

// Collect retrieves the list of loaded kernel modules from /proc/modules
// and parses them into KModConfig structures
func (s *KModCollector) Collect(ctx context.Context) ([]Configuration, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root := "/proc/modules"
	res := make([]Configuration, 0, 100)

	cmdline, err := os.ReadFile(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read KMod config: %w", err)
	}

	params := strings.Split(string(cmdline), "\n")

	for _, param := range params {
		p := strings.TrimSpace(param)
		if p == "" {
			continue
		}

		mod := strings.Split(p, " ")

		res = append(res, Configuration{
			Type: KModType,
			Data: KModConfig{
				Name: mod[0],
			},
		})
	}

	return res, nil
}
