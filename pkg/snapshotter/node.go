package snapshotter

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializers"
	"golang.org/x/sync/errgroup"
)

// NodeSnapshotter is a snapshotter that collects configuration from the current node.
type NodeSnapshotter struct {
	Factory    collectors.CollectorFactory
	Serializer serializers.Serializer
	Logger     *slog.Logger
}

// Run collects configuration from the current node and outputs it to stdout.
// It implements the Snapshotter interface.
func (n *NodeSnapshotter) Run(ctx context.Context) error {
	if n.Logger == nil {
		n.Logger = slog.Default()
	}
	if n.Factory == nil {
		n.Factory = collectors.NewDefaultCollectorFactory()
	}

	n.Logger.Info("starting node snapshot")

	// Pre-allocate with estimated capacity
	var mu sync.Mutex
	snapshot := make([]collectors.Configuration, 0, 670)

	g, ctx := errgroup.WithContext(ctx)

	// Collect kernel modules concurrently
	g.Go(func() error {
		n.Logger.Debug("collecting kernel modules")
		km := n.Factory.CreateKModCollector()
		kMod, err := km.Collect(ctx)
		if err != nil {
			n.Logger.Error("failed to collect kmod", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect kMod info: %w", err)
		}
		mu.Lock()
		snapshot = append(snapshot, kMod...)
		mu.Unlock()
		n.Logger.Debug("collected kernel modules", slog.Int("count", len(kMod)))
		return nil
	})

	// Collect systemd concurrently
	g.Go(func() error {
		n.Logger.Debug("collecting systemd services")
		sd := n.Factory.CreateSystemDCollector()
		systemd, err := sd.Collect(ctx)
		if err != nil {
			n.Logger.Error("failed to collect systemd", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect systemd info: %w", err)
		}
		mu.Lock()
		snapshot = append(snapshot, systemd...)
		mu.Unlock()
		n.Logger.Debug("collected systemd services", slog.Int("count", len(systemd)))
		return nil
	})

	// Collect grub concurrently
	g.Go(func() error {
		n.Logger.Debug("collecting grub configuration")
		g := n.Factory.CreateGrubCollector()
		grub, err := g.Collect(ctx)
		if err != nil {
			n.Logger.Error("failed to collect grub", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect grub info: %w", err)
		}
		mu.Lock()
		snapshot = append(snapshot, grub...)
		mu.Unlock()
		n.Logger.Debug("collected grub parameters", slog.Int("count", len(grub)))
		return nil
	})

	// Collect sysctl concurrently
	g.Go(func() error {
		n.Logger.Debug("collecting sysctl configuration")
		s := n.Factory.CreateSysctlCollector()
		sysctl, err := s.Collect(ctx)
		if err != nil {
			n.Logger.Error("failed to collect sysctl", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect sysctl info: %w", err)
		}
		mu.Lock()
		snapshot = append(snapshot, sysctl...)
		mu.Unlock()
		n.Logger.Debug("collected sysctl parameters", slog.Int("count", len(sysctl)))
		return nil
	})

	// Wait for all collectors to complete
	if err := g.Wait(); err != nil {
		return err
	}

	n.Logger.Info("snapshot collection complete", slog.Int("total_configs", len(snapshot)))

	// Serialize output
	if n.Serializer == nil {
		n.Serializer = serializers.NewWriter(serializers.FormatJSON, nil)
	}

	if err := n.Serializer.Serialize(snapshot); err != nil {
		n.Logger.Error("failed to serialize", slog.String("error", err.Error()))
		return fmt.Errorf("failed to serialize: %w", err)
	}

	return nil
}
