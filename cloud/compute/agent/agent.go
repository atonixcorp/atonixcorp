package agent

import (
	"atonixcorp/cloud/compute/hypervisor"
	"atonixcorp/cloud/compute/rpc"
	"fmt"
	"log"
	"time"
)

type Agent struct {
	NodeID string
	RPC    rpc.RPC
	HV     hypervisor.Hypervisor
}

func NewAgent(nodeID string, backend rpc.RPC) *Agent {
	return &Agent{
		NodeID: nodeID,
		RPC:    backend,
		HV:     hypervisor.NewLibvirtDriver(),
	}
}

func (a *Agent) Start() {
	fmt.Println("Compute Agent started on node:", a.NodeID)

	for {
		msg, err := a.RPC.Receive()
		if err != nil {
			log.Printf("compute agent receive error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if err := a.HandleMessage(msg); err != nil {
			log.Printf("compute agent message %q failed: %v", msg.Type, err)
		}
	}
}
