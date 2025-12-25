package os

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

var (
	// Keys to filter out from GRUB config for privacy/security
	filterOutGrubKeys = []string{
		"root",
	}
)

// collectGRUB retrieves the GRUB bootloader parameters from /proc/cmdline
// and returns them as a subtype with key-value pairs for each boot parameter.
func (c *Collector) collectGRUB(ctx context.Context) (*measurement.Subtype, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root := "/proc/cmdline"
	cmdline, err := os.ReadFile(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read grub config: %w", err)
	}

	// Validate UTF-8
	if !utf8.Valid(cmdline) {
		return nil, fmt.Errorf("grub config contains invalid UTF-8")
	}

	// Limit size (1MB max)
	const maxSize = 1 << 20
	if len(cmdline) > maxSize {
		return nil, fmt.Errorf("grub config exceeds maximum size of %d bytes", maxSize)
	}

	params := strings.Split(string(cmdline), " ")
	props := make(map[string]measurement.Reading, 0)

	for _, param := range params {
		p := strings.TrimSpace(param)
		if p == "" {
			continue
		}

		key, val := "", ""
		// Split on first '=' only to handle values like "root=PARTUUID=xyz"
		s := strings.SplitN(p, "=", 2)
		if len(s) == 1 {
			key = s[0]
		} else {
			key = s[0]
			val = s[1]
		}

		props[key] = measurement.Str(val)
	}

	res := &measurement.Subtype{
		Name: "grub",
		Data: measurement.FilterOut(props, filterOutGrubKeys),
	}

	return res, nil
}
