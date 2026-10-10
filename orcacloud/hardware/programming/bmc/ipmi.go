package bmc

import (
	"os/exec"
	"strings"
)

// IPMIClient wraps ipmitool calls.
type IPMIClient struct{}

// NewIPMIClient creates a new IPMI client.
func NewIPMIClient() *IPMIClient {
	return &IPMIClient{}
}

func (c *IPMIClient) ReadSensors() ([]Sensor, error) {
	out, err := exec.Command("ipmitool", "sdr").CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var sensors []Sensor

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		s := Sensor{
			Raw: line,
		}
		if len(parts) > 0 {
			s.Name = parts[0]
		}
		if len(parts) > 1 {
			s.Value = parts[1]
		}
		if len(parts) > 2 {
			s.Status = parts[2]
		}
		sensors = append(sensors, s)
	}

	return sensors, nil
}

func (c *IPMIClient) PowerCycle() error {
	_, err := exec.Command("ipmitool", "chassis", "power", "cycle").CombinedOutput()
	return err
}

func (c *IPMIClient) PowerReset() error {
	_, err := exec.Command("ipmitool", "chassis", "power", "reset").CombinedOutput()
	return err
}
