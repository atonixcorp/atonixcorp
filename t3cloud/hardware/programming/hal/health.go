package hal

// NodeHealth is the aggregated health view used by T3Cloud.
type NodeHealth struct {
	CPUUsage    float64
	MemoryUsage float64
	Disks       []DiskStatus
	Nics        []NicStats
	Sensors     []SensorReading
}

// HealthSummary aggregates all hardware signals into a single struct.
func (h *DefaultHAL) HealthSummary() (NodeHealth, error) {
	cpu, err := h.CPUUsage()
	if err != nil {
		return NodeHealth{}, err
	}

	mem, err := h.MemoryUsage()
	if err != nil {
		return NodeHealth{}, err
	}

	disks, err := h.DiskHealth()
	if err != nil {
		return NodeHealth{}, err
	}

	nics, err := h.NicStats()
	if err != nil {
		return NodeHealth{}, err
	}

	sensors, err := h.Sensors()
	if err != nil {
		return NodeHealth{}, err
	}

	return NodeHealth{
		CPUUsage:    cpu,
		MemoryUsage: mem,
		Disks:       disks,
		Nics:        nics,
		Sensors:     sensors,
	}, nil
}
