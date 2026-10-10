package hal

// HAL is the main hardware abstraction interface.
// Other packages (telemetry, control plane, agents) depend on this, not on OS/tools directly.
type HAL interface {
	CPUUsage() (float64, error)
	MemoryUsage() (float64, error)
	DiskHealth() ([]DiskStatus, error)
	NicStats() ([]NicStats, error)
	Sensors() ([]SensorReading, error)
	HealthSummary() (NodeHealth, error)
}

// DefaultHAL is the concrete implementation for Linux-based T3Cloud nodes.
type DefaultHAL struct {
	// In future: config, logger, command runner, etc.
}

// NewHAL creates a new HAL instance.
func NewHAL() HAL {
	return &DefaultHAL{}
}
