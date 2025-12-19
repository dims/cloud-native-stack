package snapshotter

import "context"

// Snapshotter is the interface that wraps the Run method.
// Run starts the snapshotter with the provided context.
type Snapshotter interface {
	Run(ctx context.Context) error
}
