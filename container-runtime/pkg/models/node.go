package models

import "time"

// Node represents a compute node in the sovereign cloud.
type Node struct {
	Name          string     `json:"name"`
	CPU           int        `json:"cpu"`
	MemoryMB      int        `json:"memory_mb"`
	UsedCPU       int        `json:"used_cpu"`
	UsedMemory    int        `json:"used_memory"`
	Status        NodeStatus `json:"status"`
	LastHeartbeat time.Time  `json:"last_heartbeat"`
}

type NodeStatus string

const (
	NodeHealthy   NodeStatus = "healthy"
	NodeUnhealthy NodeStatus = "unhealthy"
	NodeOffline   NodeStatus = "offline"
)
