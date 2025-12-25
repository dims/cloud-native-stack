package snapshotter

import "context"

// Snapshotter is the interface that wraps the Run method.
// Measure starts the snapshotter with the provided context.
type Snapshotter interface {
	Measure(ctx context.Context) error
}
