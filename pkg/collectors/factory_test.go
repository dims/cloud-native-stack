package collectors_test

import (
	"context"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
)

func TestDefaultCollectorFactory_CreateKModCollector(t *testing.T) {
	factory := collectors.NewDefaultCollectorFactory()

	collector := factory.CreateKModCollector()
	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}

	// Verify it implements Collector interface
	ctx := context.Background()
	_, err := collector.Collect(ctx)
	if err != nil {
		// Error is acceptable (file might not exist), just verify interface works
		t.Logf("Collect returned error (acceptable): %v", err)
	}
}

func TestDefaultCollectorFactory_CreateSystemDCollector(t *testing.T) {
	factory := collectors.NewDefaultCollectorFactory()
	factory.SystemDServices = []string{"test.service"}

	collector := factory.CreateSystemDCollector()
	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}

	// Verify it's configured correctly
	systemdCollector, ok := collector.(*collectors.SystemDCollector)
	if !ok {
		t.Fatal("Expected *SystemDCollector")
	}

	if len(systemdCollector.Services) != 1 || systemdCollector.Services[0] != "test.service" {
		t.Errorf("Expected [test.service], got %v", systemdCollector.Services)
	}
}

func TestDefaultCollectorFactory_CreateGrubCollector(t *testing.T) {
	factory := collectors.NewDefaultCollectorFactory()

	collector := factory.CreateGrubCollector()
	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx)
	if err != nil {
		t.Logf("Collect returned error (acceptable): %v", err)
	}
}

func TestDefaultCollectorFactory_CreateSysctlCollector(t *testing.T) {
	factory := collectors.NewDefaultCollectorFactory()

	collector := factory.CreateSysctlCollector()
	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx)
	if err != nil {
		t.Logf("Collect returned error (acceptable): %v", err)
	}
}

func TestDefaultCollectorFactory_AllCollectors(t *testing.T) {
	factory := collectors.NewDefaultCollectorFactory()

	collectorFuncs := []func() collectors.Collector{
		factory.CreateKModCollector,
		factory.CreateSystemDCollector,
		factory.CreateGrubCollector,
		factory.CreateSysctlCollector,
	}

	for i, createFunc := range collectorFuncs {
		collector := createFunc()
		if collector == nil {
			t.Errorf("Collector %d returned nil", i)
		}
	}
}
