package collector

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
	collector := &ImageCollector{Clientset: fakeClient}

	m, err := collector.Collect(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, measurement.TypeImage, m.Type)
	assert.Len(t, m.Subtypes, 1)
	assert.NotNil(t, m.Subtypes[0].Data)

	data := m.Subtypes[0].Data
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

	collector := &ImageCollector{Clientset: fake.NewClientset()}
	m, err := collector.Collect(ctx)

	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Equal(t, context.Canceled, err)
}
