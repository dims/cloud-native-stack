package collectors

import (
	"context"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestKubernetesCollector_Collect(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewClientset()
	fakeDiscovery := fakeClient.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.28.0",
		Platform:   "linux/amd64",
		GoVersion:  "go1.20.7",
	}
	collector := &KubernetesCollector{Clientset: fakeClient}

	m, err := collector.Collect(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, measurement.TypeK8s, m.Type)
	assert.Len(t, m.Subtypes, 1)
	assert.NotNil(t, m.Subtypes[0].Data)

	data := m.Subtypes[0].Data
	if assert.Len(t, data, 3) {
		if reading, ok := data["version"]; assert.True(t, ok) {
			assert.Equal(t, "v1.28.0", reading.Any())
		}
		if reading, ok := data["platform"]; assert.True(t, ok) {
			assert.Equal(t, "linux/amd64", reading.Any())
		}
		if reading, ok := data["goVersion"]; assert.True(t, ok) {
			assert.Equal(t, "go1.20.7", reading.Any())
		}
	}
}

func TestKubernetesCollector_CollectWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector := &KubernetesCollector{Clientset: fake.NewClientset()}
	m, err := collector.Collect(ctx)

	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Equal(t, context.Canceled, err)
}
