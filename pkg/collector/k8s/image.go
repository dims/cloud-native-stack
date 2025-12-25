package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// collectContainerImages extracts unique container images from all pods.
func (k *Collector) collectContainerImages(ctx context.Context) (map[string]measurement.Reading, error) {
	pods, err := k.ClientSet.CoreV1().Pods("").List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Track all locations for each image
	imageLocations := make(map[string][]string)
	recordImage := func(imageRef, location string) {
		if imageRef == "" {
			return
		}
		imageLocations[imageRef] = append(imageLocations[imageRef], location)
	}
	for _, pod := range pods.Items {
		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		locationPrefix := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)

		for _, container := range pod.Spec.Containers {
			recordImage(container.Image, fmt.Sprintf("%s:%s", locationPrefix, container.Name))
		}
		for _, container := range pod.Spec.InitContainers {
			recordImage(container.Image, fmt.Sprintf("%s:init-%s", locationPrefix, container.Name))
		}
		for _, container := range pod.Spec.EphemeralContainers {
			recordImage(container.Image, fmt.Sprintf("%s:ephemeral-%s", locationPrefix, container.Name))
		}
	}

	// Convert to final result format
	images := make(map[string]measurement.Reading)
	for imageRef, locations := range imageLocations {
		images[imageRef] = measurement.Str(strings.Join(locations, ", "))
	}

	slog.Debug("collected container images", slog.Int("count", len(images)))
	return images, nil
}
