package datacenter

import (
	"fmt"

	"atnetwork/core"
)

type VLANManager struct {
	// later: DB, switch API client, etc.
}

func NewVLANManager() *VLANManager {
	return &VLANManager{}
}

func (m *VLANManager) AllocateForNetwork(n core.Network) error {
	// In real code: pick VLAN ID, store mapping, push to switches
	fmt.Println("VLANManager: allocate VLAN for network", n.ID, n.CIDR)
	return nil
}
