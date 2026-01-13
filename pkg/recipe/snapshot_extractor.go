/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/

package recipe

import (
	"context"
	"fmt"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

// DetectionSource describes where a criteria value was detected from.
type DetectionSource struct {
	Value    string // The detected value
	Source   string // Human-readable source description
	RawValue string // The raw value from the snapshot (e.g., full version string)
}

// CriteriaDetection holds criteria with detection sources for transparency.
type CriteriaDetection struct {
	Service     *DetectionSource
	Accelerator *DetectionSource
	OS          *DetectionSource
	Intent      *DetectionSource
	Nodes       *DetectionSource
}

// constraintPath represents a parsed fully qualified constraint path.
// Format: {Type}.{Subtype}.{Key}
type constraintPath struct {
	Type    measurement.Type
	Subtype string
	Key     string
}

// parseConstraintPath parses a fully qualified constraint path.
func parseConstraintPath(path string) (*constraintPath, error) {
	if path == "" {
		return nil, fmt.Errorf("constraint path cannot be empty")
	}

	parts := strings.SplitN(path, ".", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid constraint path %q: expected format {Type}.{Subtype}.{Key}", path)
	}

	measurementType, valid := measurement.ParseType(parts[0])
	if !valid {
		return nil, fmt.Errorf("invalid measurement type %q in constraint path %q", parts[0], path)
	}

	return &constraintPath{
		Type:    measurementType,
		Subtype: parts[1],
		Key:     parts[2],
	}, nil
}

// extractValue extracts the value at this path from a snapshot.
func (cp *constraintPath) extractValue(snap *snapshotter.Snapshot) (string, error) {
	if snap == nil {
		return "", fmt.Errorf("snapshot is nil")
	}

	// Find the measurement with matching type
	var targetMeasurement *measurement.Measurement
	for _, m := range snap.Measurements {
		if m.Type == cp.Type {
			targetMeasurement = m
			break
		}
	}

	if targetMeasurement == nil {
		return "", fmt.Errorf("measurement type %q not found", cp.Type)
	}

	// Find the subtype
	var targetSubtype *measurement.Subtype
	for i := range targetMeasurement.Subtypes {
		if targetMeasurement.Subtypes[i].Name == cp.Subtype {
			targetSubtype = &targetMeasurement.Subtypes[i]
			break
		}
	}

	if targetSubtype == nil {
		return "", fmt.Errorf("subtype %q not found in measurement type %q", cp.Subtype, cp.Type)
	}

	// Find the key in data
	reading, exists := targetSubtype.Data[cp.Key]
	if !exists {
		return "", fmt.Errorf("key %q not found in subtype %q", cp.Key, cp.Subtype)
	}

	return reading.String(), nil
}

// detectionRule maps a constraint path and value pattern to a criteria field value.
type detectionRule struct {
	// Path is the constraint path to extract value from
	Path *constraintPath
	// ValuePattern is the pattern to match (exact match, or contains for patterns starting with "contains:")
	ValuePattern string
	// CriteriaField is which criteria field this rule sets (service, accelerator, os, intent)
	CriteriaField string
	// CriteriaValue is the value to set when the pattern matches
	CriteriaValue string
	// SourceDesc is the human-readable description of where this was detected from
	SourceDesc string
}

// matches checks if the extracted value matches this rule's pattern.
func (r *detectionRule) matches(extractedValue string) bool {
	// Check for contains: prefix pattern
	if pattern, found := strings.CutPrefix(r.ValuePattern, "contains:"); found {
		return containsIgnoreCase(extractedValue, pattern)
	}

	// Exact match (case-insensitive)
	return strings.EqualFold(extractedValue, r.ValuePattern)
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// getBuiltInDetectionRules returns detection rules derived from common patterns.
// These rules use constraint paths defined in recipe overlays for extraction.
func getBuiltInDetectionRules() []*detectionRule {
	var rules []*detectionRule

	// Direct service type detection from K8s.server.service field
	servicePath, _ := parseConstraintPath("K8s.server.service")
	if servicePath != nil {
		rules = append(rules,
			&detectionRule{
				Path:          servicePath,
				ValuePattern:  "eks",
				CriteriaField: "service",
				CriteriaValue: string(CriteriaServiceEKS),
				SourceDesc:    "K8s server.service field",
			},
			&detectionRule{
				Path:          servicePath,
				ValuePattern:  "gke",
				CriteriaField: "service",
				CriteriaValue: string(CriteriaServiceGKE),
				SourceDesc:    "K8s server.service field",
			},
			&detectionRule{
				Path:          servicePath,
				ValuePattern:  "aks",
				CriteriaField: "service",
				CriteriaValue: string(CriteriaServiceAKS),
				SourceDesc:    "K8s server.service field",
			},
			&detectionRule{
				Path:          servicePath,
				ValuePattern:  "oke",
				CriteriaField: "service",
				CriteriaValue: string(CriteriaServiceOKE),
				SourceDesc:    "K8s server.service field",
			},
		)
	}

	// Service detection from K8s version string (K8s.server.version constraint path)
	versionPath, _ := parseConstraintPath("K8s.server.version")
	if versionPath != nil {
		rules = append(rules,
			&detectionRule{
				Path:          versionPath,
				ValuePattern:  "contains:-eks-",
				CriteriaField: "service",
				CriteriaValue: string(CriteriaServiceEKS),
				SourceDesc:    "K8s version string",
			},
			&detectionRule{
				Path:          versionPath,
				ValuePattern:  "contains:-gke",
				CriteriaField: "service",
				CriteriaValue: string(CriteriaServiceGKE),
				SourceDesc:    "K8s version string",
			},
			&detectionRule{
				Path:          versionPath,
				ValuePattern:  "contains:-aks",
				CriteriaField: "service",
				CriteriaValue: string(CriteriaServiceAKS),
				SourceDesc:    "K8s version string",
			},
		)
	}

	// OS detection from OS.release.ID (matches OS.release.ID constraint in overlays)
	osReleasePath, _ := parseConstraintPath("OS.release.ID")
	if osReleasePath != nil {
		rules = append(rules,
			&detectionRule{
				Path:          osReleasePath,
				ValuePattern:  "ubuntu",
				CriteriaField: "os",
				CriteriaValue: string(CriteriaOSUbuntu),
				SourceDesc:    "/etc/os-release ID",
			},
			&detectionRule{
				Path:          osReleasePath,
				ValuePattern:  "rhel",
				CriteriaField: "os",
				CriteriaValue: string(CriteriaOSRHEL),
				SourceDesc:    "/etc/os-release ID",
			},
			&detectionRule{
				Path:          osReleasePath,
				ValuePattern:  "cos",
				CriteriaField: "os",
				CriteriaValue: string(CriteriaOSCOS),
				SourceDesc:    "/etc/os-release ID",
			},
			&detectionRule{
				Path:          osReleasePath,
				ValuePattern:  "amzn",
				CriteriaField: "os",
				CriteriaValue: string(CriteriaOSAmazonLinux),
				SourceDesc:    "/etc/os-release ID",
			},
			&detectionRule{
				Path:          osReleasePath,
				ValuePattern:  "amazonlinux",
				CriteriaField: "os",
				CriteriaValue: string(CriteriaOSAmazonLinux),
				SourceDesc:    "/etc/os-release ID",
			},
		)
	}

	// GPU/Accelerator detection from GPU.smi.gpu.model
	gpuModelPath, _ := parseConstraintPath("GPU.smi.gpu.model")
	if gpuModelPath != nil {
		rules = append(rules,
			&detectionRule{
				Path:          gpuModelPath,
				ValuePattern:  "contains:h100",
				CriteriaField: "accelerator",
				CriteriaValue: string(CriteriaAcceleratorH100),
				SourceDesc:    "nvidia-smi gpu.model",
			},
			&detectionRule{
				Path:          gpuModelPath,
				ValuePattern:  "contains:gb200",
				CriteriaField: "accelerator",
				CriteriaValue: string(CriteriaAcceleratorGB200),
				SourceDesc:    "nvidia-smi gpu.model",
			},
			&detectionRule{
				Path:          gpuModelPath,
				ValuePattern:  "contains:a100",
				CriteriaField: "accelerator",
				CriteriaValue: string(CriteriaAcceleratorA100),
				SourceDesc:    "nvidia-smi gpu.model",
			},
			&detectionRule{
				Path:          gpuModelPath,
				ValuePattern:  "contains:l40",
				CriteriaField: "accelerator",
				CriteriaValue: string(CriteriaAcceleratorL40),
				SourceDesc:    "nvidia-smi gpu.model",
			},
		)
	}

	// GPU detection from GPU.device.model
	gpuDevicePath, _ := parseConstraintPath("GPU.device.model")
	if gpuDevicePath != nil {
		rules = append(rules,
			&detectionRule{
				Path:          gpuDevicePath,
				ValuePattern:  "contains:h100",
				CriteriaField: "accelerator",
				CriteriaValue: string(CriteriaAcceleratorH100),
				SourceDesc:    "GPU model field",
			},
			&detectionRule{
				Path:          gpuDevicePath,
				ValuePattern:  "contains:gb200",
				CriteriaField: "accelerator",
				CriteriaValue: string(CriteriaAcceleratorGB200),
				SourceDesc:    "GPU model field",
			},
			&detectionRule{
				Path:          gpuDevicePath,
				ValuePattern:  "contains:a100",
				CriteriaField: "accelerator",
				CriteriaValue: string(CriteriaAcceleratorA100),
				SourceDesc:    "GPU model field",
			},
			&detectionRule{
				Path:          gpuDevicePath,
				ValuePattern:  "contains:l40",
				CriteriaField: "accelerator",
				CriteriaValue: string(CriteriaAcceleratorL40),
				SourceDesc:    "GPU model field",
			},
		)
	}

	return rules
}

// ExtractCriteriaFromSnapshot extracts criteria from a snapshot using detection rules
// derived from recipe constraint paths. Returns both criteria and detection sources
// for transparency.
func ExtractCriteriaFromSnapshot(_ context.Context, snap *snapshotter.Snapshot) (*Criteria, *CriteriaDetection) {
	criteria := NewCriteria()
	detection := &CriteriaDetection{}

	if snap == nil {
		return criteria, detection
	}

	// Get detection rules (derived from constraint paths used in recipe overlays)
	rules := getBuiltInDetectionRules()

	// Apply detection rules
	for _, rule := range rules {
		// Extract value from snapshot using constraint path
		value, err := rule.Path.extractValue(snap)
		if err != nil {
			continue // Skip if value not found
		}

		// Check if value matches rule pattern
		if !rule.matches(value) {
			continue
		}

		// Apply the criteria value based on field (only if not already set)
		switch rule.CriteriaField {
		case "service":
			if detection.Service == nil {
				parsed, err := ParseCriteriaServiceType(rule.CriteriaValue)
				if err == nil {
					criteria.Service = parsed
					detection.Service = &DetectionSource{
						Value:    rule.CriteriaValue,
						Source:   rule.SourceDesc,
						RawValue: value,
					}
				}
			}
		case "accelerator":
			if detection.Accelerator == nil {
				parsed, err := ParseCriteriaAcceleratorType(rule.CriteriaValue)
				if err == nil {
					criteria.Accelerator = parsed
					detection.Accelerator = &DetectionSource{
						Value:    rule.CriteriaValue,
						Source:   rule.SourceDesc,
						RawValue: value,
					}
				}
			}
		case "os":
			if detection.OS == nil {
				parsed, err := ParseCriteriaOSType(rule.CriteriaValue)
				if err == nil {
					criteria.OS = parsed
					detection.OS = &DetectionSource{
						Value:    rule.CriteriaValue,
						Source:   rule.SourceDesc,
						RawValue: value,
					}
				}
			}
		case "intent":
			if detection.Intent == nil {
				parsed, err := ParseCriteriaIntentType(rule.CriteriaValue)
				if err == nil {
					criteria.Intent = parsed
					detection.Intent = &DetectionSource{
						Value:    rule.CriteriaValue,
						Source:   rule.SourceDesc,
						RawValue: value,
					}
				}
			}
		}
	}

	return criteria, detection
}
