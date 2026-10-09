package api

// MetricsResponse contains real-time metrics for a container.
type MetricsResponse struct {
	ContainerID string  `json:"container_id"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryMB    float64 `json:"memory_mb"`
	NetworkIn   float64 `json:"network_in"`
	NetworkOut  float64 `json:"network_out"`
	Timestamp   string  `json:"timestamp"`
}
