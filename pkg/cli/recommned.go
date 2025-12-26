/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/recommender"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

func recommendCmd() *cli.Command {
	return &cli.Command{
		Name:                  "recommend",
		EnableShellCompletion: true,
		Usage:                 "Generate system configuration recommendations based on snapshot",
		Description: `Generate system configuration recommendations based on snapshot including:
  - CPU and GPU settings
  - GRUB boot parameters
  - Kubernetes cluster configuration
  - Loaded kernel modules
  - Sysctl kernel parameters
  - SystemD service configurations

The recommendation can be output in JSON, YAML, or table format.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "snapshot",
				Aliases:  []string{"f"},
				Required: true,
				Usage:    "snapshot file path",
			},
			&cli.StringFlag{
				Name:     "intent",
				Aliases:  []string{"i"},
				Value:    recipe.IntentAny.String(),
				Usage:    fmt.Sprintf("intended use case for the recommendations (%s)", recipe.SupportedIntentTypes()),
				Required: true,
			},
			outputFlag,
			formatFlag,
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Parse output format
			outFormat := serializer.Format(cmd.String("format"))
			if outFormat.IsUnknown() {
				return fmt.Errorf("unknown output format: %q", outFormat)
			}

			// Parse intent
			intentStr := cmd.String("intent")
			intent := recipe.IntentType(intentStr)
			if !intent.IsValid() {
				return fmt.Errorf("invalid intent type: %q", intentStr)
			}

			// Load snapshot
			snapFilePath := cmd.String("snapshot")
			snap, err := snapshotter.SnapshotFromFile(snapFilePath)
			if err != nil {
				return fmt.Errorf("failed to load snapshot from %q: %w", snapFilePath, err)
			}

			// Create recommender service
			service := recommender.New(
				recommender.WithVersion(version),
			)

			rec, err := service.Recommend(ctx, intent, snap)
			if err != nil {
				return fmt.Errorf("failed to generate recommendations: %w", err)
			}

			ser := serializer.NewFileWriterOrStdout(outFormat, cmd.String("output"))
			defer func() {
				if err := ser.Close(); err != nil {
					slog.Warn("failed to close serializer", "error", err)
				}
			}()

			return ser.Serialize(rec)
		},
	}
}
