package metrics

import "time"

type ContainerMetrics struct {
	ContainerID string
	CPUPercent  float64
	MemoryMB    float64
	NetworkIn   float64
	NetworkOut  float64
	Timestamp   time.Time
}

type MetricsCollector struct{}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

func (mc *MetricsCollector) Collect(containerID string) (*ContainerMetrics, error) {
	// TODO: integrate with cAdvisor or containerd metrics
	return &ContainerMetrics{
		ContainerID: containerID,
		CPUPercent:  0.5,
		MemoryMB:    128,
		NetworkIn:   10,
		NetworkOut:  5,
		Timestamp:   time.Now(),
	}, nil
}
