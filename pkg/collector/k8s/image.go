package k8s

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// collectContainerImages extracts unique container images from all pods.
func (i *Collector) collectContainerImages(ctx context.Context, k8sClient kubernetes.Interface) (map[string]measurement.Reading, error) {
	pods, err := k8sClient.CoreV1().Pods("").List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	images := make(map[string]measurement.Reading)
	recordImage := func(imageRef, location string) {
		if imageRef == "" {
			return
		}
		if _, exists := images[imageRef]; exists {
			return
		}
		images[imageRef] = measurement.Str(location)
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

	slog.Debug("collected container images", slog.Int("count", len(images)))
	return images, nil
}
