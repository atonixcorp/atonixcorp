package api

// NodeInfo describes a compute node in the cluster.
type NodeInfo struct {
	Name       string `json:"name"`
	CPU        int    `json:"cpu"`
	MemoryMB   int    `json:"memory_mb"`
	UsedCPU    int    `json:"used_cpu"`
	UsedMemory int    `json:"used_memory"`
	Status     string `json:"status"`
}

// NodesResponse returns all nodes known to the scheduler.
type NodesResponse struct {
	Nodes []NodeInfo `json:"nodes"`
}
