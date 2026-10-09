package models

import "time"

// ContainerMetrics holds real-time metrics for a running container.
type ContainerMetrics struct {
	ContainerID string    `json:"container_id"`
	CPUPercent  float64   `json:"cpu_percent"`
	MemoryMB    float64   `json:"memory_mb"`
	NetworkIn   float64   `json:"network_in"`
	NetworkOut  float64   `json:"network_out"`
	Timestamp   time.Time `json:"timestamp"`
}
