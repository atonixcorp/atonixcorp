package agent

import (
	"atonixcorp/cloud/compute/rpc"
	"fmt"
	"runtime"
)

func (a *Agent) ReportHealth() error {
	health := map[string]any{
		"node":        a.NodeID,
		"cpus":        runtime.NumCPU(),
		"go_routines": runtime.NumGoroutine(),
	}

	if err := a.RPC.Send(rpc.RPCMessage{
		Type:    "node.health",
		Payload: health,
	}); err != nil {
		return fmt.Errorf("report compute node health: %w", err)
	}

	fmt.Println("Agent: health sent")
	return nil
}
