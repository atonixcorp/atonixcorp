package bmc

// Manager is the high-level BMC interface used by the rest of the system.
type Manager interface {
	SensorStatus() ([]Sensor, error)
	PowerCycle() error
	Reboot() error
}

// DefaultManager chooses IPMI or Redfish under the hood.
type DefaultManager struct {
	ipmi    *IPMIClient
	redfish *RedfishClient
}

// NewManager creates a new BMC manager.
func NewManager() *DefaultManager {
	return &DefaultManager{
		ipmi:    NewIPMIClient(),
		redfish: NewRedfishClient(),
	}
}

func (m *DefaultManager) SensorStatus() ([]Sensor, error) {
	// For now, prefer IPMI; later you can add detection logic.
	return m.ipmi.ReadSensors()
}

func (m *DefaultManager) PowerCycle() error {
	return m.ipmi.PowerCycle()
}

func (m *DefaultManager) Reboot() error {
	return m.ipmi.PowerReset()
}
