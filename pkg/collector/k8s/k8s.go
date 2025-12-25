package k8s

import (
	"context"
	"fmt"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Collector collects information about the Kubernetes cluster.
type Collector struct {
	ClientSet  kubernetes.Interface
	RestConfig *rest.Config
}

// Collect retrieves Kubernetes cluster version information from the API server.
// This provides cluster version details for comparison across environments.
func (k *Collector) Collect(ctx context.Context) (*measurement.Measurement, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := k.getClient(); err != nil {
		return nil, err
	}
	// Cluster Version
	versions, err := k.collectServer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect server version: %w", err)
	}

	// Cluster Images
	images, err := k.collectContainerImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect container images: %w", err)
	}

	// Cluster Policies
	policies, err := k.collectClusterPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect cluster policies: %w", err)
	}

	res := &measurement.Measurement{
		Type: measurement.TypeK8s,
		Subtypes: []measurement.Subtype{
			{
				Name: "server",
				Data: versions,
			},
			{
				Name: "image",
				Data: images,
			},
			{
				Name: "policy",
				Data: policies,
			},
		},
	}

	return res, nil
}

func (k *Collector) getClient() error {
	if k.ClientSet != nil && k.RestConfig != nil {
		return nil
	}
	var err error
	k.ClientSet, k.RestConfig, err = getKubeClient("")
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}
	return nil
}
