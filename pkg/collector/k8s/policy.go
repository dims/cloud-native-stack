package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// collectClusterPolicies retrieves ClusterPolicy custom resources from all API groups and namespaces.
// It dynamically discovers all ClusterPolicy CRDs regardless of their API group.
func (k *Collector) collectClusterPolicies(ctx context.Context) (map[string]measurement.Reading, error) {
	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(k.RestConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Discover all API resources
	discoveryClient := k.ClientSet.Discovery()
	apiResourceLists, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		slog.Debug("failed to discover API resources", slog.String("error", err.Error()))
		return make(map[string]measurement.Reading), nil
	}

	policyData := make(map[string]measurement.Reading)

	// Find all ClusterPolicy resources across all API groups
	for _, apiResourceList := range apiResourceLists {
		if apiResourceList == nil {
			continue
		}

		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			continue
		}

		for _, resource := range apiResourceList.APIResources {
			// Look for ClusterPolicy kind
			if resource.Kind != "ClusterPolicy" {
				continue
			}

			// Skip subresources (they contain a slash like "clusterpolicies/status")
			if len(resource.Name) == 0 || strings.Contains(resource.Name, "/") {
				continue
			}

			// Construct GVR for this ClusterPolicy resource
			gvr := schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: resource.Name,
			}

			slog.Debug("found clusterpolicy resource",
				slog.String("group", gv.Group),
				slog.String("version", gv.Version),
				slog.String("resource", resource.Name))

			// List all instances across all namespaces
			policies, err := dynamicClient.Resource(gvr).Namespace("").List(ctx, v1.ListOptions{})
			if err != nil {
				slog.Debug("failed to list clusterpolicy",
					slog.String("group", gv.Group),
					slog.String("error", err.Error()))
				continue
			}

			// Process each policy
			for _, policy := range policies.Items {
				// Check for context cancellation
				if err := ctx.Err(); err != nil {
					return nil, err
				}

				// Create a unique key with group prefix to avoid conflicts
				policyKey := fmt.Sprintf("%s/%s", gv.Group, policy.GetName())
				if policy.GetNamespace() != "" {
					policyKey = fmt.Sprintf("%s/%s/%s", gv.Group, policy.GetNamespace(), policy.GetName())
				}

				// Extract spec as JSON for detailed information
				spec, found, err := unstructured.NestedMap(policy.Object, "spec")
				if err != nil || !found {
					slog.Warn("failed to extract spec from clusterpolicy",
						slog.String("name", policyKey),
						slog.String("error", fmt.Sprintf("%v", err)))
					continue
				}

				// Convert spec to JSON string for storage
				specJSON, err := json.Marshal(spec)
				if err != nil {
					slog.Warn("failed to marshal clusterpolicy spec",
						slog.String("name", policyKey),
						slog.String("error", err.Error()))
					continue
				}

				policyData[policyKey] = measurement.Str(string(specJSON))
			}
		}
	}

	slog.Debug("collected cluster policies", slog.Int("count", len(policyData)))
	return policyData, nil
}
