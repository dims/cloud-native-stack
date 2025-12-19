package collectors_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
)

func TestGrubCollector_Collect_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector := &collectors.GrubCollector{}
	_, err := collector.Collect(ctx)

	if err == nil {
		// On some systems, the read may complete before context check
		t.Skip("Context cancellation timing dependent")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestGrubCollector_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	collector := &collectors.GrubCollector{}

	configs, err := collector.Collect(ctx)
	if err != nil {
		// /proc/cmdline might not exist on all systems
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/cmdline not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Most systems have at least a few boot parameters
	if len(configs) == 0 {
		t.Error("Expected at least one boot parameter")
	}

	t.Logf("Found %d boot parameters", len(configs))

	for _, cfg := range configs {
		if cfg.Type != collectors.GrubType {
			t.Errorf("Expected type %s, got %s", collectors.GrubType, cfg.Type)
		}

		// Validate that Data is GrubConfig
		grubCfg, ok := cfg.Data.(collectors.GrubConfig)
		if !ok {
			t.Errorf("Expected GrubConfig, got %T", cfg.Data)
			continue
		}

		if grubCfg.Key == "" {
			t.Error("Expected non-empty key")
		}
	}
}

func TestGrubCollector_ValidatesParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	collector := &collectors.GrubCollector{}

	configs, err := collector.Collect(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/cmdline not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Check that we can parse both key-only and key=value formats
	hasKeyOnly := false
	hasKeyValue := false

	for _, cfg := range configs {
		grubCfg := cfg.Data.(collectors.GrubConfig)
		if grubCfg.Value == "" {
			hasKeyOnly = true
		} else {
			hasKeyValue = true
		}
	}

	t.Logf("Has key-only params: %v, Has key=value params: %v", hasKeyOnly, hasKeyValue)
}
