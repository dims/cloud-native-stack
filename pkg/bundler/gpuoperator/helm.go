package gpuoperator

import (
	"fmt"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const (
	strTrue = "true"
)

// ConfigValue holds a configuration value with its context/explanation.
type ConfigValue struct {
	Value   interface{}
	Context string // Human-readable explanation from recipe
}

// HelmValues represents the data structure for GPU Operator Helm values.
type HelmValues struct {
	Timestamp                     string
	DriverRegistry                ConfigValue
	GPUOperatorVersion            ConfigValue
	EnableDriver                  ConfigValue
	DriverVersion                 ConfigValue
	UseOpenKernelModule           ConfigValue
	NvidiaContainerToolkitVersion ConfigValue
	DevicePluginVersion           ConfigValue
	DCGMVersion                   ConfigValue
	DCGMExporterVersion           ConfigValue
	MIGStrategy                   ConfigValue
	EnableGDS                     ConfigValue
	VGPULicenseServer             ConfigValue
	EnableSecureBoot              ConfigValue
	CustomLabels                  map[string]string
	Namespace                     string
}

// GenerateHelmValues generates Helm values from a recipe.
func GenerateHelmValues(recipe *recipe.Recipe, config map[string]string) *HelmValues {
	values := &HelmValues{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		DriverRegistry:   ConfigValue{Value: getConfigValue(config, "driver_registry", "nvcr.io/nvidia")},
		EnableDriver:     ConfigValue{Value: true},
		MIGStrategy:      ConfigValue{Value: "single"},
		EnableGDS:        ConfigValue{Value: false},
		EnableSecureBoot: ConfigValue{Value: false},
		CustomLabels:     make(map[string]string),
		Namespace:        getConfigValue(config, "namespace", "gpu-operator"),
	}

	// Extract GPU Operator configuration from recipe measurements
	for _, m := range recipe.Measurements {
		switch m.Type {
		case measurement.TypeK8s:
			values.extractK8sSettings(m)
		case measurement.TypeGPU:
			values.extractGPUSettings(m)
		case measurement.TypeSystemD, measurement.TypeOS:
			// Not used for Helm values generation
		}
	}

	// Apply config overrides
	values.applyConfigOverrides(config)

	return values
}

// extractK8sSettings extracts Kubernetes-related settings from measurements.
func (v *HelmValues) extractK8sSettings(m *measurement.Measurement) {
	for _, st := range m.Subtypes {
		// Extract context for this subtype
		subtypeContext := getSubtypeContext(st.Context)

		// Extract version information from 'image' subtype
		if st.Name == "image" {
			if val, ok := st.Data["gpu-operator"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := getFieldContext(st.Context, "gpu-operator", subtypeContext)
					v.GPUOperatorVersion = ConfigValue{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["driver"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := getFieldContext(st.Context, "driver", subtypeContext)
					v.DriverVersion = ConfigValue{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["container-toolkit"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := getFieldContext(st.Context, "container-toolkit", subtypeContext)
					v.NvidiaContainerToolkitVersion = ConfigValue{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["k8s-device-plugin"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := getFieldContext(st.Context, "k8s-device-plugin", subtypeContext)
					v.DevicePluginVersion = ConfigValue{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["dcgm"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := getFieldContext(st.Context, "dcgm", subtypeContext)
					v.DCGMVersion = ConfigValue{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["dcgm-exporter"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := getFieldContext(st.Context, "dcgm-exporter", subtypeContext)
					v.DCGMExporterVersion = ConfigValue{Value: s, Context: ctx}
				}
			}
		}

		// Extract configuration flags from 'config' subtype
		if st.Name == "config" {
			// MIG configuration (boolean in recipe)
			if val, ok := st.Data["mig"]; ok {
				if b, ok := val.Any().(bool); ok && b {
					ctx := getFieldContext(st.Context, "mig", subtypeContext)
					v.MIGStrategy = ConfigValue{Value: "mixed", Context: ctx}
				}
			}
			// UseOpenKernelModule (camelCase in recipe)
			if val, ok := st.Data["useOpenKernelModule"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := getFieldContext(st.Context, "useOpenKernelModule", subtypeContext)
					v.UseOpenKernelModule = ConfigValue{Value: b, Context: ctx}
				}
			}
			// RDMA support (affects GDS)
			if val, ok := st.Data["rdma"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := getFieldContext(st.Context, "rdma", subtypeContext)
					v.EnableGDS = ConfigValue{Value: b, Context: ctx}
				}
			}
		}
	}
}

// extractGPUSettings extracts GPU-related settings from measurements.
func (v *HelmValues) extractGPUSettings(m *measurement.Measurement) {
	for _, st := range m.Subtypes {
		subtypeContext := getSubtypeContext(st.Context)

		// Recipe uses 'smi' subtype for nvidia-smi output
		if st.Name == "smi" {
			if val, ok := st.Data["driver-version"]; ok {
				if s, ok := val.Any().(string); ok {
					// Only set if not already set from K8s measurements
					if cv, ok := v.DriverVersion.Value.(string); !ok || cv == "" {
						ctx := getFieldContext(st.Context, "driver-version", subtypeContext)
						v.DriverVersion = ConfigValue{Value: s, Context: ctx}
					}
				}
			}
		}
	}
}

// applyConfigOverrides applies configuration overrides to values.
func (v *HelmValues) applyConfigOverrides(config map[string]string) {
	if val, ok := config["driver_version"]; ok && val != "" {
		v.DriverVersion = ConfigValue{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["gpu_operator_version"]; ok && val != "" {
		v.GPUOperatorVersion = ConfigValue{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["mig_strategy"]; ok && val != "" {
		v.MIGStrategy = ConfigValue{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_gds"]; ok {
		v.EnableGDS = ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["vgpu_license_server"]; ok && val != "" {
		v.VGPULicenseServer = ConfigValue{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["namespace"]; ok && val != "" {
		v.Namespace = val
	}

	// Custom labels
	for k, val := range config {
		if len(k) > 6 && k[:6] == "label_" {
			v.CustomLabels[k[6:]] = val
		}
	}
}

// getSubtypeContext extracts the general context from subtype context map.
func getSubtypeContext(contextMap map[string]string) string {
	if desc, ok := contextMap["description"]; ok && desc != "" {
		return desc
	}
	if reason, ok := contextMap["reason"]; ok && reason != "" {
		return reason
	}
	return ""
}

// getFieldContext gets the context for a specific field, falling back to subtype context.
func getFieldContext(contextMap map[string]string, fieldName, subtypeContext string) string {
	// Try field-specific context first (e.g., "gpu-operator-context")
	if ctx, ok := contextMap[fieldName+"-context"]; ok && ctx != "" {
		return ctx
	}
	if ctx, ok := contextMap[fieldName]; ok && ctx != "" {
		return ctx
	}
	// Fall back to subtype-level context
	return subtypeContext
}

// getConfigValue gets a value from config with a default fallback.
func getConfigValue(config map[string]string, key, defaultValue string) string {
	if val, ok := config[key]; ok && val != "" {
		return val
	}
	return defaultValue
}

// ToMap converts HelmValues to a map for template rendering.
func (v *HelmValues) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"Timestamp":                     v.Timestamp,
		"DriverRegistry":                v.DriverRegistry,
		"GPUOperatorVersion":            v.GPUOperatorVersion,
		"EnableDriver":                  v.EnableDriver,
		"DriverVersion":                 v.DriverVersion,
		"UseOpenKernelModule":           v.UseOpenKernelModule,
		"NvidiaContainerToolkitVersion": v.NvidiaContainerToolkitVersion,
		"DevicePluginVersion":           v.DevicePluginVersion,
		"DCGMVersion":                   v.DCGMVersion,
		"DCGMExporterVersion":           v.DCGMExporterVersion,
		"MIGStrategy":                   v.MIGStrategy,
		"EnableGDS":                     v.EnableGDS,
		"VGPULicenseServer":             v.VGPULicenseServer,
		"EnableSecureBoot":              v.EnableSecureBoot,
		"CustomLabels":                  v.CustomLabels,
		"Namespace":                     v.Namespace,
	}
}

// Validate validates the Helm values.
func (v *HelmValues) Validate() error {
	if v.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if dr, ok := v.DriverRegistry.Value.(string); !ok || dr == "" {
		return fmt.Errorf("driver registry cannot be empty")
	}
	if ms, ok := v.MIGStrategy.Value.(string); ok {
		if ms != "single" && ms != "mixed" {
			return fmt.Errorf("invalid MIG strategy: %s (must be single or mixed)", ms)
		}
	}
	return nil
}
