package snapshotter

import (
	"fmt"

	"github.com/NVIDIA/cloud-native-stack/cli/pkg/collectors"
	"github.com/NVIDIA/cloud-native-stack/cli/pkg/serilizers"
)

type NodeSnapshotter struct {
}

func (n *NodeSnapshotter) Run(config any) error {
	fmt.Println("snapshotting current node")

	snapshot := []collectors.Configuraion{}

	km := collectors.KModCollector{}
	kMod, err := km.Collect(nil)
	if err != nil {
		return fmt.Errorf("failed to collect kMod info: %w", err)
	}
	snapshot = append(snapshot, kMod...)

	sd := collectors.SystemDCollector{}
	systemd, err := sd.Collect(nil)
	if err != nil {
		return fmt.Errorf("failed to collect systemd info: %w", err)
	}
	snapshot = append(snapshot, systemd...)

	g := collectors.GrubCollector{}
	grub, err := g.Collect(nil)
	if err != nil {
		return fmt.Errorf("failed to collect grub info: %w", err)
	}
	snapshot = append(snapshot, grub...)

	s := collectors.SysctlCollector{}
	sysctl, err := s.Collect(nil)
	if err != nil {
		return fmt.Errorf("failed to collect sysctl info: %w", err)
	}
	snapshot = append(snapshot, sysctl...)

	stdout := serilizers.StdoutSerilizer{}
	err = stdout.Serilize(snapshot)
	if err != nil {
		return fmt.Errorf("failed to serilize: %w", err)
	}

	return nil
}
