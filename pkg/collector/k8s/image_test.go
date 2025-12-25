package k8s

import (
	"context"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

// Helper function to create a test collector with fake client
func createTestCollector(objects ...corev1.Pod) *Collector {
	runtimeObjects := make([]runtime.Object, len(objects))
	for i := range objects {
		runtimeObjects[i] = &objects[i]
	}
	fakeClient := fake.NewClientset(runtimeObjects...)
	// Set a fake server version to avoid nil pointer
	fakeDiscovery := fakeClient.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.28.0",
		Platform:   "linux/amd64",
		GoVersion:  "go1.20.7",
	}
	// Set RestConfig to avoid getClient() trying to connect to real cluster
	return &Collector{
		ClientSet:  fakeClient,
		RestConfig: &rest.Config{},
	}
}

func TestImageCollector_Collect(t *testing.T) {
	ctx := context.Background()
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-a", Namespace: "ns"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "c1", Image: "repo/image:tag"},
			},
			InitContainers: []corev1.Container{
				{Name: "init", Image: "repo/init:latest"},
			},
		},
	}
	collector := createTestCollector(pod)

	m, err := collector.Collect(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, measurement.TypeK8s, m.Type)
	// Should have 3 subtypes: server, image, and policy
	assert.Len(t, m.Subtypes, 3)

	// Find the image subtype
	var imageSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == "image" {
			imageSubtype = &m.Subtypes[i]
			break
		}
	}
	if !assert.NotNil(t, imageSubtype, "Expected to find image subtype") {
		return
	}

	data := imageSubtype.Data
	if assert.Len(t, data, 2) {
		reading, ok := data["image:tag"]
		if assert.True(t, ok) {
			assert.Equal(t, "ns/pod-a:c1", reading.Any())
		}
		initReading, ok := data["init:latest"]
		if assert.True(t, ok) {
			assert.Equal(t, "ns/pod-a:init-init", initReading.Any())
		}
	}
}

func TestImageCollector_CollectWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector := createTestCollector()
	m, err := collector.Collect(ctx)

	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Equal(t, context.Canceled, err)
}

func TestImageCollector_MultipleLocations(t *testing.T) {
	ctx := context.Background()
	pod1 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "ns1"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "web", Image: "nginx:1.21"},
			},
		},
	}
	pod2 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-2", Namespace: "ns2"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "nginx:1.21"},
			},
		},
	}
	collector := createTestCollector(pod1, pod2)

	m, err := collector.Collect(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, m)

	// Find the image subtype
	var imageSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == "image" {
			imageSubtype = &m.Subtypes[i]
			break
		}
	}
	if !assert.NotNil(t, imageSubtype) {
		return
	}

	data := imageSubtype.Data
	reading, ok := data["nginx:1.21"]
	if assert.True(t, ok) {
		locations := reading.Any().(string)
		// Should contain both locations
		assert.Contains(t, locations, "ns1/pod-1:web")
		assert.Contains(t, locations, "ns2/pod-2:app")
	}
}
