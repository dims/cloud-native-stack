package collector

import (
	"github.com/NVIDIA/cloud-native-stack/pkg/collector/gpu"
	"github.com/NVIDIA/cloud-native-stack/pkg/collector/k8s"
	"github.com/NVIDIA/cloud-native-stack/pkg/collector/os"
	"github.com/NVIDIA/cloud-native-stack/pkg/collector/systemd"
)

// Factory creates collectors with their dependencies.
// This interface enables dependency injection for testing.
type Factory interface {
	CreateSystemDCollector() Collector
	CreateOSCollector() Collector
	CreateKubernetesCollector() Collector
	CreateGPUCollector() Collector
}

// DefaultFactory creates collectors with production dependencies.
type DefaultFactory struct {
	SystemDServices []string
}

// NewDefaultFactory creates a factory with default settings.
func NewDefaultFactory() *DefaultFactory {
	return &DefaultFactory{
		SystemDServices: []string{
			"containerd.service",
			"docker.service",
			"kubelet.service",
		},
	}
}

// CreateSMICollector creates an GPU collector.
func (f *DefaultFactory) CreateGPUCollector() Collector {
	return &gpu.Collector{}
}

// CreateSystemDCollector creates a systemd collector.
func (f *DefaultFactory) CreateSystemDCollector() Collector {
	return &systemd.Collector{
		Services: f.SystemDServices,
	}
}

// CreateGrubCollector creates a GRUB collector.
func (f *DefaultFactory) CreateOSCollector() Collector {
	return &os.Collector{}
}

// CreateKubernetesCollector creates a Kubernetes API collector.
func (f *DefaultFactory) CreateKubernetesCollector() Collector {
	return &k8s.Collector{}
}
