package node

import (
	"fmt"
	"os/exec"

	"atnetwork/core"
)

type OVSManager struct {
	BridgeName string
}

func NewOVSManager() *OVSManager {
	return &OVSManager{
		BridgeName: "br-int",
	}
}

func (m *OVSManager) EnsurePort(p core.Port) error {
	fmt.Printf("[NODE OVS] EnsurePort: ID=%s NetworkID=%s DeviceID=%s MAC=%s\n",
		p.ID, p.NetworkID, p.DeviceID, p.MAC)

	// Create OVS port
	if err := exec.Command("ovs-vsctl", "add-port", m.BridgeName, p.ID).Run(); err != nil {
		return err
	}

	// Set external-ids or other metadata if needed
	_ = exec.Command("ovs-vsctl", "set", "Interface", p.ID,
		fmt.Sprintf("external-ids:atn-port-id=%s", p.ID)).Run()

	return nil
}
