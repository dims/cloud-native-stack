package snapshotter

import (
	"context"

	"github.com/NVIDIA/cloud-native-stack/pkg/header"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

// Snapshotter is the interface that wraps the Run method.
// Measure starts the snapshotter with the provided context.
type Snapshotter interface {
	Measure(ctx context.Context) error
}

// Snapshot represents the collected configuration snapshot of a node.
type Snapshot struct {
	header.Header `json:",inline" yaml:",inline"`

	// Measurements contains the collected measurements from various collectors.
	Measurements []*measurement.Measurement `json:"measurements" yaml:"measurements"`
}
