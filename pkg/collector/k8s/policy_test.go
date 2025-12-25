package k8s

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/dynamic/fake"
	fakeclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

// Helper to create a mock ClusterPolicy
func createMockClusterPolicy(name, namespace string) *unstructured.Unstructured {
	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "nvidia.com/v1",
			"kind":       "ClusterPolicy",
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{
				"operator": map[string]interface{}{
					"defaultRuntime": "containerd",
				},
				"driver": map[string]interface{}{
					"enabled": true,
					"version": "550.54.15",
				},
			},
		},
	}
	if namespace != "" {
		policy.SetNamespace(namespace)
	}
	return policy
}

// Helper to create test collector with mocked clients
func createTestPolicyCollector(policies ...*unstructured.Unstructured) (*Collector, error) {
	// Create fake kubernetes client
	fakeK8sClient := fakeclient.NewClientset()
	fakeDiscovery := fakeK8sClient.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.28.0",
		Platform:   "linux/amd64",
		GoVersion:  "go1.20.7",
	}

	// Set up API resources for ClusterPolicy
	clusterPolicyResource := metav1.APIResource{
		Name:         "clusterpolicies",
		SingularName: "clusterpolicy",
		Namespaced:   false,
		Kind:         "ClusterPolicy",
	}

	fakeDiscovery.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "nvidia.com/v1",
			APIResources: []metav1.APIResource{clusterPolicyResource},
		},
	}

	// Create a simple rest config for dynamic client
	restConfig := &rest.Config{}

	return &Collector{
		ClientSet:  fakeK8sClient,
		RestConfig: restConfig,
	}, nil
}

func TestPolicyCollector_Collect(t *testing.T) {
	// This test validates the structure when policies are found
	// Note: The dynamic client discovery is complex to mock fully,
	// so we're testing the parsing logic and structure
	ctx := context.Background()

	collector, err := createTestPolicyCollector()
	assert.NoError(t, err)

	m, err := collector.Collect(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, measurement.TypeK8s, m.Type)

	// Find the policy subtype
	var policySubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == "policy" {
			policySubtype = &m.Subtypes[i]
			break
		}
	}
	assert.NotNil(t, policySubtype, "Expected to find policy subtype")
	// With our mock setup, we expect empty policies since we can't fully mock dynamic client
	assert.NotNil(t, policySubtype.Data)
}

func TestPolicyCollector_EmptyCluster(t *testing.T) {
	// Test behavior when no policies exist
	ctx := context.Background()

	collector, err := createTestPolicyCollector()
	assert.NoError(t, err)

	policies, err := collector.collectClusterPolicies(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, policies)
	assert.Empty(t, policies, "Expected no policies in empty cluster")
}

func TestPolicyCollector_WithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector, err := createTestPolicyCollector()
	assert.NoError(t, err)

	m, err := collector.Collect(ctx)
	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Equal(t, context.Canceled, err)
}

func TestPolicyCollector_ParsesClusterPolicySpec(t *testing.T) {
	// Test the spec parsing logic in isolation
	policy := createMockClusterPolicy("test-policy", "")

	// Extract spec
	spec, found, err := unstructured.NestedMap(policy.Object, "spec")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.NotNil(t, spec)

	// Verify spec contains expected fields
	operator, found, err := unstructured.NestedMap(spec, "operator")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "containerd", operator["defaultRuntime"])

	driver, found, err := unstructured.NestedMap(spec, "driver")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, true, driver["enabled"])
	assert.Equal(t, "550.54.15", driver["version"])
}

func TestPolicyCollector_SpecSerialization(t *testing.T) {
	// Test that spec can be properly serialized to JSON
	policy := createMockClusterPolicy("test-policy", "")

	spec, found, err := unstructured.NestedMap(policy.Object, "spec")
	assert.NoError(t, err)
	assert.True(t, found)

	// Serialize to JSON
	specJSON, err := json.Marshal(spec)
	assert.NoError(t, err)
	assert.NotEmpty(t, specJSON)

	// Deserialize and verify
	var deserializedSpec map[string]interface{}
	err = json.Unmarshal(specJSON, &deserializedSpec)
	assert.NoError(t, err)
	assert.NotNil(t, deserializedSpec)
}

// Integration test helper
func createFakeDynamicClient(objects ...runtime.Object) *fake.FakeDynamicClient {
	s := scheme.Scheme
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "nvidia.com", Version: "v1", Kind: "ClusterPolicy"},
		&unstructured.Unstructured{},
	)

	return fake.NewSimpleDynamicClient(s, objects...)
}

// Helper for discovery mock
func createMockDiscovery() discovery.DiscoveryInterface {
	fakeClient := fakeclient.NewClientset()
	fakeDiscovery := fakeClient.Discovery().(*fakediscovery.FakeDiscovery)

	// Set up API resources for ClusterPolicy
	clusterPolicyResource := metav1.APIResource{
		Name:         "clusterpolicies",
		SingularName: "clusterpolicy",
		Namespaced:   false,
		Kind:         "ClusterPolicy",
	}

	fakeDiscovery.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "nvidia.com/v1",
			APIResources: []metav1.APIResource{clusterPolicyResource},
		},
	}

	return fakeDiscovery
}
