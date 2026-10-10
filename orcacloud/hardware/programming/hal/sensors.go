package hal

import (
	"os/exec"
	"strings"
)

// SensorReading represents a single sensor value from BMC/IPMI.
type SensorReading struct {
	Name   string
	Value  string
	Status string
	Raw    string
}

// Sensors uses ipmitool sdr to read sensor data.
// Later you can replace this with native Redfish.
func (h *DefaultHAL) Sensors() ([]SensorReading, error) {
	out, err := exec.Command("ipmitool", "sdr").CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var result []SensorReading

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		r := SensorReading{
			Raw: line,
		}
		if len(parts) > 0 {
			r.Name = parts[0]
		}
		if len(parts) > 1 {
			r.Value = parts[1]
		}
		if len(parts) > 2 {
			r.Status = parts[2]
		}
		result = append(result, r)
	}

	return result, nil
}
