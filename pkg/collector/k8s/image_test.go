package k8s

import (
	"context"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestImageCollector_Collect(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-a", Namespace: "ns"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "c1", Image: "repo/image:tag"},
				},
				InitContainers: []corev1.Container{
					{Name: "init", Image: "repo/init:latest"},
				},
			},
		},
	)
	collector := &Collector{ClientSet: fakeClient}

	m, err := collector.Collect(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, measurement.TypeK8s, m.Type)
	// Should have 2 subtypes: server and image
	assert.Len(t, m.Subtypes, 2)

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
		reading, ok := data["repo/image:tag"]
		if assert.True(t, ok) {
			assert.Equal(t, "ns/pod-a:c1", reading.Any())
		}
		initReading, ok := data["repo/init:latest"]
		if assert.True(t, ok) {
			assert.Equal(t, "ns/pod-a:init-init", initReading.Any())
		}
	}
}

func TestImageCollector_CollectWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector := &Collector{ClientSet: fake.NewClientset()}
	m, err := collector.Collect(ctx)

	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Equal(t, context.Canceled, err)
}
