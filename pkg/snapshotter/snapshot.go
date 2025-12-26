package snapshotter

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/collector"
	"github.com/NVIDIA/cloud-native-stack/pkg/collector/k8s"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"

	"golang.org/x/sync/errgroup"
)

// NodeSnapshotter collects system configuration measurements from the current node.
// It coordinates multiple collectors in parallel to gather data about Kubernetes,
// GPU hardware, OS configuration, and systemd services, then serializes the results.
type NodeSnapshotter struct {
	// Version is the snapshotter version.
	Version string

	// Factory is the collector factory to use. If nil, the default factory is used.
	Factory collector.Factory

	// Serializer is the serializer to use for output. If nil, a default stdout JSON serializer is used.
	Serializer serializer.Serializer
}

// Measure collects configuration measurements from the current node and serializes the snapshot.
// It runs collectors in parallel using errgroup for efficient data gathering.
// If any collector fails, the entire operation returns an error.
// The resulting snapshot is serialized using the configured Serializer.
func (n *NodeSnapshotter) Measure(ctx context.Context) error {
	if n.Factory == nil {
		n.Factory = collector.NewDefaultFactory()
	}

	slog.Debug("starting node snapshot")

	// Track overall snapshot collection duration
	start := time.Now()
	defer func() {
		snapshotCollectionDuration.Observe(time.Since(start).Seconds())
	}()

	// Pre-allocate with estimated capacity
	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)

	// Initialize snapshot structure
	snap := NewSnapshot()

	// Collect metadata
	g.Go(func() error {
		collectorStart := time.Now()
		defer func() {
			snapshotCollectorDuration.WithLabelValues("metadata").Observe(time.Since(collectorStart).Seconds())
		}()
		nodeName := k8s.GetNodeName()
		mu.Lock()
		snap.Set("Snapshot")
		snap.Metadata["snapshot-version"] = n.Version
		snap.Metadata["source-node"] = nodeName
		mu.Unlock()
		slog.Debug("obtained node metadata", slog.String("name", nodeName), slog.String("version", n.Version))
		return nil
	})

	// Collect Kubernetes configuration
	g.Go(func() error {
		collectorStart := time.Now()
		defer func() {
			snapshotCollectorDuration.WithLabelValues("k8s").Observe(time.Since(collectorStart).Seconds())
		}()
		slog.Debug("collecting kubernetes resources")
		kc := n.Factory.CreateKubernetesCollector()
		k8sResources, err := kc.Collect(ctx)
		if err != nil {
			slog.Error("failed to collect kubernetes resources", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect kubernetes resources: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, k8sResources)
		mu.Unlock()
		return nil
	})

	// Collect SystemD services
	g.Go(func() error {
		collectorStart := time.Now()
		defer func() {
			snapshotCollectorDuration.WithLabelValues("systemd").Observe(time.Since(collectorStart).Seconds())
		}()
		slog.Debug("collecting systemd services")
		sd := n.Factory.CreateSystemDCollector()
		systemd, err := sd.Collect(ctx)
		if err != nil {
			slog.Error("failed to collect systemd", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect systemd info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, systemd)
		mu.Unlock()
		return nil
	})

	// Collect OS
	g.Go(func() error {
		collectorStart := time.Now()
		defer func() {
			snapshotCollectorDuration.WithLabelValues("os").Observe(time.Since(collectorStart).Seconds())
		}()
		slog.Debug("collecting OS configuration")
		g := n.Factory.CreateOSCollector()
		grub, err := g.Collect(ctx)
		if err != nil {
			slog.Error("failed to collect OS", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect OS info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, grub)
		mu.Unlock()
		return nil
	})

	// Collect GPU
	g.Go(func() error {
		collectorStart := time.Now()
		defer func() {
			snapshotCollectorDuration.WithLabelValues("gpu").Observe(time.Since(collectorStart).Seconds())
		}()
		slog.Debug("collecting GPU configuration")
		smi := n.Factory.CreateGPUCollector()
		smiConfigs, err := smi.Collect(ctx)
		if err != nil {
			slog.Error("failed to collect GPU", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect SMI info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, smiConfigs)
		mu.Unlock()
		return nil
	})

	// Wait for all collectors to complete
	if err := g.Wait(); err != nil {
		snapshotCollectionTotal.WithLabelValues("error").Inc()
		return err
	}

	snapshotCollectionTotal.WithLabelValues("success").Inc()
	snapshotMeasurementCount.Set(float64(len(snap.Measurements)))

	slog.Debug("snapshot collection complete", slog.Int("total_configs", len(snap.Measurements)))

	// Serialize output
	if n.Serializer == nil {
		n.Serializer = serializer.NewStdoutWriter(serializer.FormatJSON)
	}

	if err := n.Serializer.Serialize(snap); err != nil {
		slog.Error("failed to serialize", slog.String("error", err.Error()))
		return fmt.Errorf("failed to serialize: %w", err)
	}

	return nil
}

// SnapshotFromFile loads a Snapshot from the specified file path.
func SnapshotFromFile(path string) (*Snapshot, error) {
	fileFormat := serializer.FormatFromPath(path)
	slog.Debug("determined snapshot file format",
		slog.String("path", path),
		slog.String("format", string(fileFormat)),
	)

	ser, err := serializer.NewFileReader(fileFormat, path)
	if err != nil {
		slog.Error("failed to create file reader", "error", err, "path", path, "format", fileFormat)
		return nil, fmt.Errorf("failed to create serializer for %q: %w", path, err)
	}

	if ser == nil {
		slog.Error("reader is unexpectedly nil despite no error")
		return nil, fmt.Errorf("reader is nil for %q", path)
	}

	defer func() {
		if ser != nil {
			if closeErr := ser.Close(); closeErr != nil {
				slog.Warn("failed to close serializer", "error", closeErr)
			}
		}
	}()

	var snap Snapshot
	if err := ser.Deserialize(&snap); err != nil {
		return nil, fmt.Errorf("failed to deserialize snapshot from %q: %w", path, err)
	}

	slog.Debug("successfully loaded snapshot from file",
		slog.String("path", path),
		slog.String("kind", snap.Kind),
		slog.String("apiVersion", snap.APIVersion),
		slog.Int("measurements", len(snap.Measurements)),
	)

	return &snap, nil
}
