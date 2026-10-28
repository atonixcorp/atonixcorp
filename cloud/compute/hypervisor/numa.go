package hypervisor

import "fmt"

type NUMANode struct {
	VMID     string
	ID       int
	CPUs     []int
	MemoryMB int
}

func PlaceNUMA(vmID string, node NUMANode) error {
	fmt.Printf("numa: placing VM %s on NUMA node %d (cpus=%v mem=%dMB)\n",
		vmID, node.ID, node.CPUs, node.MemoryMB)
	// TODO: write to cpuset + memory nodes
	return nil
}
