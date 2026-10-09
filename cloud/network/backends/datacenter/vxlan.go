package datacenter

import (
	"fmt"

	"atnetwork/core"
)

type VXLANManager struct {
	// later: DB, switch API client, etc.
}

func NewVXLANManager() *VXLANManager {
	return &VXLANManager{}
}

func (m *VXLANManager) AllocateForNetwork(n core.Network) error {
	// In real code: pick VNI, store mapping, push to switches
	fmt.Println("VXLANManager: allocate VNI for network", n.ID, n.CIDR)
	return nil
}
