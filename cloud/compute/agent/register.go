package agent

import (
	"atonixcorp/cloud/compute/rpc"
	"fmt"
)

func (a *Agent) Register() error {
	if err := a.RPC.Send(rpc.RPCMessage{
		Type: "node.register",
		Payload: map[string]any{
			"node_id": a.NodeID,
		},
	}); err != nil {
		return fmt.Errorf("register compute node: %w", err)
	}

	fmt.Println("Agent: registered with control plane")
	return nil
}
