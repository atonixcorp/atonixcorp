package hal

import (
	"os/exec"
	"strings"
)

// NicStats represents basic NIC statistics.
type NicStats struct {
	Interface string
	Raw       string
}

// NicHealth is derived from stats; you can expand later.
func (h *DefaultHAL) NicStats() ([]NicStats, error) {
	ifaces := []string{"eth0", "eth1"}
	var result []NicStats

	for _, iface := range ifaces {
		out, err := exec.Command("ethtool", "-S", iface).CombinedOutput()
		if err != nil {
			// ignore missing interfaces
			continue
		}
		result = append(result, NicStats{
			Interface: iface,
			Raw:       strings.TrimSpace(string(out)),
		})
	}

	return result, nil
}
