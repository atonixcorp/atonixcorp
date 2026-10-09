package node

import (
	"fmt"
	"os"

	"atnetwork/core"
)

type DHCPManager struct {
	ConfigDir string
}

func NewDHCPManager(dir string) *DHCPManager {
	return &DHCPManager{ConfigDir: dir}
}

func (m *DHCPManager) EnsureLease(p core.Port) error {
	fmt.Printf("[NODE DHCP] EnsureLease: PortID=%s DeviceID=%s\n", p.ID, p.DeviceID)

	// In real code: look up IP from DB (port_ips table)
	// Here we just write a placeholder config per port
	cfg := fmt.Sprintf("# DHCP lease for port %s\n", p.ID)
	path := fmt.Sprintf("%s/%s.conf", m.ConfigDir, p.ID)

	return os.WriteFile(path, []byte(cfg), 0644)
}
