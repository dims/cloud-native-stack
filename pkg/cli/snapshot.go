/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializers"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"

	"github.com/spf13/cobra"
)

var (
	outputFormat    string
	systemdServices []string
)

// snapshotCmd represents the snapshot command
var snapshotCmd = &cobra.Command{
	Use:     "snapshot",
	Aliases: []string{"snap"},
	GroupID: "functional",
	Short:   "Capture system configuration snapshot",
	Long: `Capture a comprehensive snapshot of system configuration including:
  - Loaded kernel modules
  - SystemD service configurations
  - GRUB boot parameters
  - Sysctl kernel parameters

The snapshot can be output in JSON, YAML, or table format.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		logger := GetLogger()

		// Parse output format
		format := serializers.Format(outputFormat)
		if format != serializers.FormatJSON &&
			format != serializers.FormatYAML &&
			format != serializers.FormatTable {
			format = serializers.FormatJSON
		}

		// Create factory with configured services
		factory := &collectors.DefaultCollectorFactory{
			SystemDServices: systemdServices,
		}

		// Create and run snapshotter
		ns := snapshotter.NodeSnapshotter{
			Version:    version,
			Factory:    factory,
			Serializer: serializers.NewWriter(format, nil),
			Logger:     logger,
		}

		return ns.Run(ctx)
	},
}

func init() {
	rootCmd.AddCommand(snapshotCmd)

	snapshotCmd.Flags().StringVarP(&outputFormat, "output", "o", "json",
		"output format (json, yaml, table)")
	snapshotCmd.Flags().StringSliceVar(&systemdServices, "systemd-services",
		[]string{"containerd.service", "docker.service", "kubelet.service"},
		"systemd services to snapshot")
}
