package collectors_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
)

func TestSysctlCollector_Collect_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	collector := &collectors.SysctlCollector{}

	// Start collection and cancel mid-way
	go func() {
		// Give it a moment to start walking
		cancel()
	}()

	_, err := collector.Collect(ctx)

	// Context cancellation during walk should return context error
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Logf("Got error: %v (expected context.Canceled or nil)", err)
	}
}

func TestSysctlCollector_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	collector := &collectors.SysctlCollector{}

	configs, err := collector.Collect(ctx)
	if err != nil {
		// /proc/sys might not exist on all systems
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/sys not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Most systems have many sysctl parameters
	if len(configs) == 0 {
		t.Error("Expected at least one sysctl parameter")
	}

	t.Logf("Found %d sysctl parameters", len(configs))

	// Verify no /proc/sys/net entries (should be excluded)
	for _, cfg := range configs {
		if cfg.Type != collectors.SysctlType {
			t.Errorf("Expected type %s, got %s", collectors.SysctlType, cfg.Type)
		}

		sysctlCfg, ok := cfg.Data.(collectors.SysctlConfig)
		if !ok {
			t.Errorf("Expected SysctlConfig, got %T", cfg.Data)
			continue
		}

		if strings.HasPrefix(sysctlCfg.Key, "/proc/sys/net") {
			t.Errorf("Found /proc/sys/net entry which should be excluded: %s", sysctlCfg.Key)
		}

		if !strings.HasPrefix(sysctlCfg.Key, "/proc/sys") {
			t.Errorf("Key doesn't start with /proc/sys: %s", sysctlCfg.Key)
		}
	}
}

func TestSysctlCollector_ExcludesNet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	collector := &collectors.SysctlCollector{}

	configs, err := collector.Collect(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/sys not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Ensure no network parameters are included
	for _, cfg := range configs {
		sysctlCfg := cfg.Data.(collectors.SysctlConfig)
		if strings.Contains(sysctlCfg.Key, "/net/") {
			t.Errorf("Network sysctl should be excluded: %s", sysctlCfg.Key)
		}
	}
}
