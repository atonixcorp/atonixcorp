package telemetry

import (
	"time"

	"t3cloud/hardware/programming/hal"
)

// Collector pulls metrics from HAL and converts them into telemetry snapshots.
type Collector struct {
	hal hal.HAL
}

// NewCollector creates a new telemetry collector.
func NewCollector(h hal.HAL) *Collector {
	return &Collector{hal: h}
}

// CollectSnapshot returns a high-level snapshot of node telemetry.
func (c *Collector) CollectSnapshot() (Snapshot, error) {
	cpu, err := c.hal.CPUUsage()
	if err != nil {
		return Snapshot{}, err
	}

	mem, err := c.hal.MemoryUsage()
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		CPUUsage:    cpu,
		MemoryUsage: mem,
	}, nil
}

// CollectMetrics returns a list of raw metrics suitable for export.
func (c *Collector) CollectMetrics() ([]Metric, error) {
	snap, err := c.CollectSnapshot()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	metrics := []Metric{
		{
			Name:      "node_cpu_usage_percent",
			Value:     snap.CPUUsage,
			Timestamp: now,
			Labels:    map[string]string{},
		},
		{
			Name:      "node_memory_usage_percent",
			Value:     snap.MemoryUsage,
			Timestamp: now,
			Labels:    map[string]string{},
		},
	}

	return metrics, nil
}
