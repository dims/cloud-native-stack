/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

// cliDetectionSource wraps recipe.DetectionSource with CLI-specific fields.
type cliDetectionSource struct {
	*recipe.DetectionSource
	Overridden bool // Whether this was overridden by CLI flag
}

// cliCriteriaDetection holds criteria detection with CLI-specific override tracking.
type cliCriteriaDetection struct {
	Service     *cliDetectionSource
	Accelerator *cliDetectionSource
	OS          *cliDetectionSource
	Intent      *cliDetectionSource
	Nodes       *cliDetectionSource
}

// PrintDetection outputs the detected criteria to the given writer.
func (cd *cliCriteriaDetection) PrintDetection(w io.Writer) {
	fmt.Fprintln(w, "Detected criteria from snapshot:")
	printDetectionField(w, "service", cd.Service)
	printDetectionField(w, "accelerator", cd.Accelerator)
	printDetectionField(w, "os", cd.OS)
	printDetectionField(w, "intent", cd.Intent)
	printDetectionField(w, "nodes", cd.Nodes)
	fmt.Fprintln(w)
}

func printDetectionField(w io.Writer, name string, ds *cliDetectionSource) {
	if ds == nil || ds.DetectionSource == nil {
		fmt.Fprintf(w, "  %-12s (not detected)\n", name+":")
		return
	}

	switch {
	case ds.Overridden:
		fmt.Fprintf(w, "  %-12s %-12s (overridden by --%s flag)\n", name+":", ds.Value, name)
	case ds.RawValue != "" && ds.RawValue != ds.Value:
		fmt.Fprintf(w, "  %-12s %-12s (from %s: %s)\n", name+":", ds.Value, ds.Source, ds.RawValue)
	default:
		fmt.Fprintf(w, "  %-12s %-12s (from %s)\n", name+":", ds.Value, ds.Source)
	}
}

// wrapDetection wraps recipe.CriteriaDetection into CLI-specific detection with override tracking.
func wrapDetection(d *recipe.CriteriaDetection) *cliCriteriaDetection {
	if d == nil {
		return &cliCriteriaDetection{}
	}
	return &cliCriteriaDetection{
		Service:     wrapSource(d.Service),
		Accelerator: wrapSource(d.Accelerator),
		OS:          wrapSource(d.OS),
		Intent:      wrapSource(d.Intent),
		Nodes:       wrapSource(d.Nodes),
	}
}

func wrapSource(s *recipe.DetectionSource) *cliDetectionSource {
	if s == nil {
		return nil
	}
	return &cliDetectionSource{DetectionSource: s}
}

func recipeCmd() *cli.Command {
	return &cli.Command{
		Name:                  "recipe",
		EnableShellCompletion: true,
		Usage:                 "Generate configuration recipe for a given set of environment parameters.",
		Description: `Generate configuration recipe based on specified environment parameters including:
  - Kubernetes service type (eks, gke, aks, oke, self-managed)
  - Accelerator type (h100, gb200, a100, l40)
  - Workload intent (training, inference)
  - GPU node operating system (ubuntu, rhel, cos, amazonlinux)
  - Number of GPU nodes in the cluster

The recipe returns a list of components with deployment order based on dependencies.
Output can be in JSON or YAML format.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "service",
				Usage: fmt.Sprintf("Kubernetes service type (e.g. %s)", strings.Join(recipe.GetCriteriaServiceTypes(), ", ")),
			},
			&cli.StringFlag{
				Name:    "accelerator",
				Aliases: []string{"gpu"},
				Usage:   fmt.Sprintf("Accelerator/GPU type (e.g. %s)", strings.Join(recipe.GetCriteriaAcceleratorTypes(), ", ")),
			},
			&cli.StringFlag{
				Name:  "intent",
				Usage: fmt.Sprintf("Workload intent (e.g. %s)", strings.Join(recipe.GetCriteriaIntentTypes(), ", ")),
			},
			&cli.StringFlag{
				Name:  "os",
				Usage: fmt.Sprintf("Operating system type of the GPU node (e.g. %s)", strings.Join(recipe.GetCriteriaOSTypes(), ", ")),
			},
			&cli.IntFlag{
				Name:  "nodes",
				Usage: "Number of worker/GPU nodes in the cluster",
			},
			&cli.StringFlag{
				Name:    "snapshot",
				Aliases: []string{"f"},
				Usage: `Path/URI to previously generated configuration snapshot.
	Supports: file paths, HTTP/HTTPS URLs, or ConfigMap URIs (cm://namespace/name).
	If provided, criteria are extracted from the snapshot.`,
			},
			outputFlag,
			formatFlag,
			kubeconfigFlag,
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Parse output format
			outFormat := serializer.Format(cmd.String("format"))
			if outFormat.IsUnknown() {
				return fmt.Errorf("unknown output format: %q", outFormat)
			}

			// Create builder
			builder := recipe.NewBuilder(
				recipe.WithVersion(version),
			)

			var result *recipe.RecipeResult
			var err error

			// Check if using snapshot
			snapFilePath := cmd.String("snapshot")
			if snapFilePath != "" {
				slog.Info("loading snapshot from", "uri", snapFilePath)
				snap, loadErr := serializer.FromFileWithKubeconfig[snapshotter.Snapshot](snapFilePath, cmd.String("kubeconfig"))
				if loadErr != nil {
					return fmt.Errorf("failed to load snapshot from %q: %w", snapFilePath, loadErr)
				}

				// Extract criteria from snapshot using detection rules derived from recipe constraints
				criteria, recipeDetection := recipe.ExtractCriteriaFromSnapshot(ctx, snap)

				// Wrap detection for CLI-specific override tracking
				detection := wrapDetection(recipeDetection)

				// Apply CLI overrides and track them
				if applyErr := applyCriteriaOverrides(cmd, criteria, detection); applyErr != nil {
					return applyErr
				}

				// Print detected criteria for transparency (to stderr so it doesn't interfere with output)
				detection.PrintDetection(cmd.ErrWriter)

				slog.Info("building recipe from snapshot", "criteria", criteria.String())
				result, err = builder.BuildFromCriteria(ctx, criteria)
			} else {
				// Build criteria from CLI flags
				criteria, buildErr := buildCriteriaFromCmd(cmd)
				if buildErr != nil {
					return fmt.Errorf("error parsing criteria: %w", buildErr)
				}

				slog.Info("building recipe from criteria", "criteria", criteria.String())
				result, err = builder.BuildFromCriteria(ctx, criteria)
			}

			if err != nil {
				return fmt.Errorf("error building recipe: %w", err)
			}

			// Serialize output
			output := cmd.String("output")
			ser := serializer.NewFileWriterOrStdout(outFormat, output)
			defer func() {
				if closer, ok := ser.(interface{ Close() error }); ok {
					if err := closer.Close(); err != nil {
						slog.Warn("failed to close serializer", "error", err)
					}
				}
			}()

			if err := ser.Serialize(ctx, result); err != nil {
				return fmt.Errorf("failed to serialize recipe: %w", err)
			}

			slog.Info("recipe generation completed",
				"output", output,
				"components", len(result.ComponentRefs),
				"overlays", len(result.Metadata.AppliedOverlays))

			return nil
		},
	}
}

// buildCriteriaFromCmd constructs a recipe.Criteria from CLI command flags.
func buildCriteriaFromCmd(cmd *cli.Command) (*recipe.Criteria, error) {
	var opts []recipe.CriteriaOption

	if s := cmd.String("service"); s != "" {
		opts = append(opts, recipe.WithCriteriaService(s))
	}
	if s := cmd.String("accelerator"); s != "" {
		opts = append(opts, recipe.WithCriteriaAccelerator(s))
	}
	if s := cmd.String("intent"); s != "" {
		opts = append(opts, recipe.WithCriteriaIntent(s))
	}
	if s := cmd.String("os"); s != "" {
		opts = append(opts, recipe.WithCriteriaOS(s))
	}
	if n := cmd.Int("nodes"); n > 0 {
		opts = append(opts, recipe.WithCriteriaNodes(n))
	}

	return recipe.BuildCriteria(opts...)
}

// applyCriteriaOverrides applies CLI flag overrides to criteria and tracks them in detection.
func applyCriteriaOverrides(cmd *cli.Command, criteria *recipe.Criteria, detection *cliCriteriaDetection) error {
	if s := cmd.String("service"); s != "" {
		parsed, err := recipe.ParseCriteriaServiceType(s)
		if err != nil {
			return err
		}
		wasDetected := detection.Service != nil
		criteria.Service = parsed
		detection.Service = &cliDetectionSource{
			DetectionSource: &recipe.DetectionSource{
				Value:  string(parsed),
				Source: "--service flag",
			},
			Overridden: wasDetected,
		}
	}
	if s := cmd.String("accelerator"); s != "" {
		parsed, err := recipe.ParseCriteriaAcceleratorType(s)
		if err != nil {
			return err
		}
		wasDetected := detection.Accelerator != nil
		criteria.Accelerator = parsed
		detection.Accelerator = &cliDetectionSource{
			DetectionSource: &recipe.DetectionSource{
				Value:  string(parsed),
				Source: "--accelerator flag",
			},
			Overridden: wasDetected,
		}
	}
	if s := cmd.String("intent"); s != "" {
		parsed, err := recipe.ParseCriteriaIntentType(s)
		if err != nil {
			return err
		}
		wasDetected := detection.Intent != nil
		criteria.Intent = parsed
		detection.Intent = &cliDetectionSource{
			DetectionSource: &recipe.DetectionSource{
				Value:  string(parsed),
				Source: "--intent flag",
			},
			Overridden: wasDetected,
		}
	}
	if s := cmd.String("os"); s != "" {
		parsed, err := recipe.ParseCriteriaOSType(s)
		if err != nil {
			return err
		}
		wasDetected := detection.OS != nil
		criteria.OS = parsed
		detection.OS = &cliDetectionSource{
			DetectionSource: &recipe.DetectionSource{
				Value:  string(parsed),
				Source: "--os flag",
			},
			Overridden: wasDetected,
		}
	}
	if n := cmd.Int("nodes"); n > 0 {
		wasDetected := detection.Nodes != nil
		criteria.Nodes = n
		detection.Nodes = &cliDetectionSource{
			DetectionSource: &recipe.DetectionSource{
				Value:  fmt.Sprintf("%d", n),
				Source: "--nodes flag",
			},
			Overridden: wasDetected,
		}
	}
	return nil
}
