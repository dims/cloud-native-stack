package networkoperator

import (
	"fmt"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/common"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const (
	strTrue = "true"
)

// HelmValues represents the data structure for Network Operator Helm values.
type HelmValues struct {
	Timestamp              string
	NetworkOperatorVersion common.ConfigValue
	OFEDVersion            common.ConfigValue
	EnableRDMA             common.ConfigValue
	EnableSRIOV            common.ConfigValue
	EnableHostDevice       common.ConfigValue
	EnableIPAM             common.ConfigValue
	EnableMultus           common.ConfigValue
	EnableWhereabouts      common.ConfigValue
	DeployOFED             common.ConfigValue
	NicType                common.ConfigValue
	ContainerRuntimeSocket common.ConfigValue
	CustomLabels           map[string]string
	Namespace              string
}

// GenerateHelmValues generates Helm values from a recipe.
func GenerateHelmValues(recipe *recipe.Recipe, config map[string]string) *HelmValues {
	values := &HelmValues{
		Timestamp:              time.Now().UTC().Format(time.RFC3339),
		EnableRDMA:             common.ConfigValue{Value: false},
		EnableSRIOV:            common.ConfigValue{Value: false},
		EnableHostDevice:       common.ConfigValue{Value: true},
		EnableIPAM:             common.ConfigValue{Value: true},
		EnableMultus:           common.ConfigValue{Value: true},
		EnableWhereabouts:      common.ConfigValue{Value: true},
		DeployOFED:             common.ConfigValue{Value: false},
		NicType:                common.ConfigValue{Value: "ConnectX"},
		ContainerRuntimeSocket: common.ConfigValue{Value: "/var/run/containerd/containerd.sock"},
		CustomLabels:           common.ExtractCustomLabels(config),
		Namespace:              common.GetConfigValue(config, "namespace", "nvidia-network-operator"),
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
		subtypeContext := common.GetSubtypeContext(st.Context)

		// Extract version information from 'image' subtype
		if st.Name == "image" {
			if val, ok := st.Data["network-operator"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "network-operator", subtypeContext)
					v.NetworkOperatorVersion = common.ConfigValue{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["ofed-driver"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "ofed-driver", subtypeContext)
					v.OFEDVersion = common.ConfigValue{Value: s, Context: ctx}
				}
			}
		}

		// Extract configuration flags from 'config' subtype
		if st.Name == "config" {
			// RDMA configuration
			if val, ok := st.Data["rdma"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "rdma", subtypeContext)
					v.EnableRDMA = common.ConfigValue{Value: b, Context: ctx}
				}
			}
			// SR-IOV configuration
			if val, ok := st.Data["sr-iov"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "sr-iov", subtypeContext)
					v.EnableSRIOV = common.ConfigValue{Value: b, Context: ctx}
				}
			}
			// OFED deployment
			if val, ok := st.Data["deploy-ofed"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "deploy-ofed", subtypeContext)
					v.DeployOFED = common.ConfigValue{Value: b, Context: ctx}
				}
			}
			// Host device plugin
			if val, ok := st.Data["host-device"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "host-device", subtypeContext)
					v.EnableHostDevice = common.ConfigValue{Value: b, Context: ctx}
				}
			}
			// IPAM plugin
			if val, ok := st.Data["ipam"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "ipam", subtypeContext)
					v.EnableIPAM = common.ConfigValue{Value: b, Context: ctx}
				}
			}
			// Multus CNI
			if val, ok := st.Data["multus"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "multus", subtypeContext)
					v.EnableMultus = common.ConfigValue{Value: b, Context: ctx}
				}
			}
			// Whereabouts IPAM
			if val, ok := st.Data["whereabouts"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "whereabouts", subtypeContext)
					v.EnableWhereabouts = common.ConfigValue{Value: b, Context: ctx}
				}
			}
			// NIC type
			if val, ok := st.Data["nic-type"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "nic-type", subtypeContext)
					v.NicType = common.ConfigValue{Value: s, Context: ctx}
				}
			}
		}

		// Extract container runtime from 'server' subtype
		if st.Name == "server" {
			if val, ok := st.Data["container-runtime"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "container-runtime", subtypeContext)
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
					v.ContainerRuntimeSocket = common.ConfigValue{Value: socket, Context: ctx}
				}
			}
		}
	}
}

// applyConfigOverrides applies configuration overrides to values.
func (v *HelmValues) applyConfigOverrides(config map[string]string) {
	if val, ok := config["network_operator_version"]; ok && val != "" {
		v.NetworkOperatorVersion = common.ConfigValue{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["ofed_version"]; ok && val != "" {
		v.OFEDVersion = common.ConfigValue{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_rdma"]; ok {
		v.EnableRDMA = common.ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_sriov"]; ok {
		v.EnableSRIOV = common.ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["deploy_ofed"]; ok {
		v.DeployOFED = common.ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_host_device"]; ok {
		v.EnableHostDevice = common.ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_ipam"]; ok {
		v.EnableIPAM = common.ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_multus"]; ok {
		v.EnableMultus = common.ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_whereabouts"]; ok {
		v.EnableWhereabouts = common.ConfigValue{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["nic_type"]; ok && val != "" {
		v.NicType = common.ConfigValue{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["container_runtime_socket"]; ok && val != "" {
		v.ContainerRuntimeSocket = common.ConfigValue{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["namespace"]; ok && val != "" {
		v.Namespace = val
	}

	// Custom labels
	v.CustomLabels = common.ExtractCustomLabels(config)
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
