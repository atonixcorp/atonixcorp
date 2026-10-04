package telemetry

import "time"

// Metric is a single telemetry datapoint.
type Metric struct {
	Name      string
	Value     float64
	Timestamp time.Time
	Labels    map[string]string
}

// Snapshot is a grouped view of node metrics.
type Snapshot struct {
	CPUUsage    float64
	MemoryUsage float64
	// You can extend with disk, NIC, etc. if you want aggregated numbers.
}
