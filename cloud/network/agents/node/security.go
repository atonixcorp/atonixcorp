package node

import (
	"fmt"
	"os/exec"

	"atnetwork/core"
)

type SecurityManager struct{}

func NewSecurityManager() *SecurityManager {
	return &SecurityManager{}
}

func (m *SecurityManager) EnsureSecurityGroups(p core.Port) error {
	fmt.Printf("[NODE SEC] EnsureSecurityGroups: PortID=%s\n", p.ID)

	// In real code:
	// - Fetch SGs attached to this port from DB (port_security_groups)
	// - Fetch rules for each SG
	// - Program iptables/nftables accordingly

	// Example placeholder: allow all for now
	cmd := exec.Command("iptables", "-A", "FORWARD", "-m", "comment",
		"--comment", fmt.Sprintf("ATN allow all for port %s", p.ID),
		"-j", "ACCEPT")

	return cmd.Run()
}
