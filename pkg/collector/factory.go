package collector

import (
	"github.com/NVIDIA/cloud-native-stack/pkg/collector/gpu"
	"github.com/NVIDIA/cloud-native-stack/pkg/collector/grub"
	"github.com/NVIDIA/cloud-native-stack/pkg/collector/k8s"
	"github.com/NVIDIA/cloud-native-stack/pkg/collector/kmod"
	"github.com/NVIDIA/cloud-native-stack/pkg/collector/sysctl"
	"github.com/NVIDIA/cloud-native-stack/pkg/collector/systemd"
)

// Factory creates collectors with their dependencies.
// This interface enables dependency injection for testing.
type Factory interface {
	CreateKModCollector() Collector
	CreateSystemDCollector() Collector
	CreateGrubCollector() Collector
	CreateSysctlCollector() Collector
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

// ComponentCollector creates a component collector.
func (f *DefaultFactory) CreateKModCollector() Collector {
	return &kmod.Collector{}
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
func (f *DefaultFactory) CreateGrubCollector() Collector {
	return &grub.Collector{}
}

// CreateSysctlCollector creates a sysctl collector.
func (f *DefaultFactory) CreateSysctlCollector() Collector {
	return &sysctl.Collector{}
}

// CreateKubernetesCollector creates a Kubernetes API collector.
func (f *DefaultFactory) CreateKubernetesCollector() Collector {
	return &k8s.Collector{}
}
