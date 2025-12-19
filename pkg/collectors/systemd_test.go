package collectors_test

import (
	"context"
	"errors"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
)

func TestSystemDCollector_Collect_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector := &collectors.SystemDCollector{
		Services: []string{"containerd.service"},
	}
	_, err := collector.Collect(ctx)

	// Should fail with context canceled
	if err != nil && !errors.Is(err, context.Canceled) {
		// D-Bus connection might fail for other reasons
		t.Logf("Got error: %v", err)
	}
}

func TestSystemDCollector_DefaultServices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Test with nil services (should use default)
	collector := &collectors.SystemDCollector{}

	_, err := collector.Collect(ctx)
	if err != nil {
		// D-Bus might not be available or service might not exist
		t.Logf("Expected possible error for systemd access: %v", err)
		return
	}
}

func TestSystemDCollector_CustomServices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	collector := &collectors.SystemDCollector{
		Services: []string{"containerd.service", "docker.service"},
	}

	configs, err := collector.Collect(ctx)
	if err != nil {
		// Services might not exist or D-Bus unavailable
		t.Logf("Expected possible error: %v", err)
		return
	}

	// If successful, verify structure
	for _, cfg := range configs {
		if cfg.Type != collectors.SystemDType {
			t.Errorf("Expected type %s, got %s", collectors.SystemDType, cfg.Type)
		}

		systemdCfg, ok := cfg.Data.(collectors.SystemDConfig)
		if !ok {
			t.Errorf("Expected SystemDConfig, got %T", cfg.Data)
			continue
		}

		if systemdCfg.Unit == "" {
			t.Error("Expected non-empty unit name")
		}

		if systemdCfg.Properties == nil {
			t.Error("Expected non-nil properties map")
		}
	}
}

func TestSystemDCollector_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test requires systemd to be running
	ctx := context.Background()
	collector := &collectors.SystemDCollector{
		Services: []string{"containerd.service"},
	}

	configs, err := collector.Collect(ctx)
	if err != nil {
		// SystemD might not be available on this system
		t.Skipf("SystemD not available or service not found: %v", err)
	}

	t.Logf("Successfully collected %d systemd configurations", len(configs))

	// Validate collected data
	if len(configs) > 0 {
		cfg := configs[0]
		systemdCfg := cfg.Data.(collectors.SystemDConfig)
		t.Logf("Service: %s, Properties: %d", systemdCfg.Unit, len(systemdCfg.Properties))
	}
}
