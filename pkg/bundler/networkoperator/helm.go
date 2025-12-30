package networkoperator

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

// HelmValues represents the data structure for Network Operator Helm values.
type HelmValues struct {
	Timestamp              string
	NetworkOperatorVersion ConfigValue
	OFEDVersion            ConfigValue
	EnableRDMA             ConfigValue
	EnableSRIOV            ConfigValue
	EnableHostDevice       ConfigValue
	EnableIPAM             ConfigValue
	EnableMultus           ConfigValue
	EnableWhereabouts      ConfigValue
	DeployOFED             ConfigValue
	NicType                ConfigValue
	ContainerRuntimeSocket ConfigValue
	CustomLabels           map[string]string
	Namespace              string
}

// GenerateHelmValues generates Helm values from a recipe.
func GenerateHelmValues(recipe *recipe.Recipe, config map[string]string) *HelmValues {
	values := &HelmValues{
		Timestamp:              time.Now().UTC().Format(time.RFC3339),
		EnableRDMA:             ConfigValue{Value: false},
		EnableSRIOV:            ConfigValue{Value: false},
		EnableHostDevice:       ConfigValue{Value: true},
		EnableIPAM:             ConfigValue{Value: true},
		EnableMultus:           ConfigValue{Value: true},
		EnableWhereabouts:      ConfigValue{Value: true},
		DeployOFED:             ConfigValue{Value: false},
		NicType:                ConfigValue{Value: "ConnectX"},
		ContainerRuntimeSocket: ConfigValue{Value: "/var/run/containerd/containerd.sock"},
		CustomLabels:           make(map[string]string),
		Namespace:              getConfigValue(config, "namespace", "nvidia-network-operator"),
	}

	// Extract Network Operator configuration from recipe measurements
	for _, m := range recipe.Measurements {
		switch m.Type {
		case measurement.TypeK8s:
			values.extractK8sSettings(m)
		case measurement.TypeSystemD, measurement.TypeOS, measurement.TypeGPU:
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
			if val, ok := st.Data["network-operator"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := getFieldContext(st.Context, "network-operator", subtypeContext)
					v.NetworkOperatorVersion = ConfigValue{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["ofed-driver"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := getFieldContext(st.Context, "ofed-driver", subtypeContext)
					v.OFEDVersion = ConfigValue{Value: s, Context: ctx}
				}
			}
		}

		// Extract configuration flags from 'config' subtype
		if st.Name == "config" {
			// RDMA configuration
			if val, ok := st.Data["rdma"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := getFieldContext(st.Context, "rdma", subtypeContext)
					v.EnableRDMA = ConfigValue{Value: b, Context: ctx}
				}
			}
			// SR-IOV configuration
			if val, ok := st.Data["sr-iov"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := getFieldContext(st.Context, "sr-iov", subtypeContext)
					v.EnableSRIOV = ConfigValue{Value: b, Context: ctx}
				}
			}
			// OFED deployment
			if val, ok := st.Data["deploy-ofed"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := getFieldContext(st.Context, "deploy-ofed", subtypeContext)
					v.DeployOFED = ConfigValue{Value: b, Context: ctx}
				}
			}
			// Host device plugin
			if val, ok := st.Data["host-device"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := getFieldContext(st.Context, "host-device", subtypeContext)
					v.EnableHostDevice = ConfigValue{Value: b, Context: ctx}
				}
			}
			// IPAM plugin
			if val, ok := st.Data["ipam"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := getFieldContext(st.Context, "ipam", subtypeContext)
					v.EnableIPAM = ConfigValue{Value: b, Context: ctx}
				}
			}
			// Multus CNI
			if val, ok := st.Data["multus"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := getFieldContext(st.Context, "multus", subtypeContext)
					v.EnableMultus = ConfigValue{Value: b, Context: ctx}
				}
			}
			// Whereabouts IPAM
			if val, ok := st.Data["whereabouts"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := getFieldContext(st.Context, "whereabouts", subtypeContext)
					v.EnableWhereabouts = ConfigValue{Value: b, Context: ctx}
				}
			}
			// NIC type
			if val, ok := st.Data["nic-type"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := getFieldContext(st.Context, "nic-type", subtypeContext)
					v.NicType = ConfigValue{Value: s, Context: ctx}
				}
			}
		}

		// Extract container runtime from 'server' subtype
		if st.Name == "server" {
			if val, ok := st.Data["container-runtime"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := getFieldContext(st.Context, "container-runtime", subtypeContext)
					var socket string
					switch s {
					case "containerd":
						socket = "/var/run/containerd/containerd.sock"
					case "docker":
						socket = "/var/run/docker.sock"
					case "cri-o":
						socket = "/var/run/crio/crio.sock"
					default:
						socket = "/var/run/containerd/containerd.sock"
					}
					v.ContainerRuntimeSocket = ConfigValue{Value: socket, Context: ctx}
				}
			}
		}
	}
}

// applyConfigOverrides applies configuration overrides to values.
func (v *HelmValues) applyConfigOverrides(config map[string]string) {
	if val, ok := config["network_operator_version"]; ok && val != "" {
		v.NetworkOperatorVersion = ConfigValue{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["ofed_version"]; ok && val != "" {
		v.OFEDVersion = ConfigValue{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_rdma"]; ok {
		v.EnableRDMA = ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_sriov"]; ok {
		v.EnableSRIOV = ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["deploy_ofed"]; ok {
		v.DeployOFED = ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_host_device"]; ok {
		v.EnableHostDevice = ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_ipam"]; ok {
		v.EnableIPAM = ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_multus"]; ok {
		v.EnableMultus = ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_whereabouts"]; ok {
		v.EnableWhereabouts = ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["nic_type"]; ok && val != "" {
		v.NicType = ConfigValue{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["container_runtime_socket"]; ok && val != "" {
		v.ContainerRuntimeSocket = ConfigValue{Value: val, Context: "Override from bundler configuration"}
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
	// Try field-specific context first (e.g., "network-operator-context")
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
		"Timestamp":              v.Timestamp,
		"NetworkOperatorVersion": v.NetworkOperatorVersion,
		"OFEDVersion":            v.OFEDVersion,
		"EnableRDMA":             v.EnableRDMA,
		"EnableSRIOV":            v.EnableSRIOV,
		"EnableHostDevice":       v.EnableHostDevice,
		"EnableIPAM":             v.EnableIPAM,
		"EnableMultus":           v.EnableMultus,
		"EnableWhereabouts":      v.EnableWhereabouts,
		"DeployOFED":             v.DeployOFED,
		"NicType":                v.NicType,
		"ContainerRuntimeSocket": v.ContainerRuntimeSocket,
		"CustomLabels":           v.CustomLabels,
		"Namespace":              v.Namespace,
	}
}

// Validate validates the Helm values.
func (v *HelmValues) Validate() error {
	if v.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if nt, ok := v.NicType.Value.(string); !ok || nt == "" {
		return fmt.Errorf("NIC type cannot be empty")
	}
	if crs, ok := v.ContainerRuntimeSocket.Value.(string); !ok || crs == "" {
		return fmt.Errorf("container runtime socket cannot be empty")
	}
	return nil
}
