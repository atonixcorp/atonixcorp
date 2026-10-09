package node

import (
	"fmt"
	"time"

	"atnetwork/core"
)

type NodeAgent struct {
	NodeID   string
	Store    core.Store
	OVS      *OVSManager
	DHCP     *DHCPManager
	Security *SecurityManager
}

func NewNodeAgent(nodeID string, store core.Store) *NodeAgent {
	return &NodeAgent{
		NodeID:   nodeID,
		Store:    store,
		OVS:      NewOVSManager(),
		DHCP:     NewDHCPManager("/etc/dnsmasq.d"),
		Security: NewSecurityManager(),
	}
}

func (a *NodeAgent) Start() {
	go func() {
		for {
			if err := a.syncOnce(); err != nil {
				fmt.Println("NodeAgent sync error:", err)
			}
			time.Sleep(15 * time.Second)
		}
	}()
}

func (a *NodeAgent) syncOnce() error {
	// Fetch desired ports from ATN DB
	ports, err := a.Store.ListPorts()
	if err != nil {
		return err
	}

	for _, p := range ports {
		// In real code: filter ports that belong to this node (via device_id or host binding)
		if err := a.OVS.EnsurePort(p); err != nil {
			fmt.Println("OVS ensure port failed:", err)
		}

		if err := a.DHCP.EnsureLease(p); err != nil {
			fmt.Println("DHCP ensure lease failed:", err)
		}

		if err := a.Security.EnsureSecurityGroups(p); err != nil {
			fmt.Println("Security ensure SG failed:", err)
		}
	}

	return nil
}
