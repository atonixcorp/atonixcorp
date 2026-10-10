package hal

import (
	"os/exec"
	"strings"
)

// DiskStatus represents health of a single disk.
type DiskStatus struct {
	Device string
	Health string
	Raw    string
}

// DiskHealth runs smartctl on common devices.
// You can extend this to discover all disks dynamically.
func (h *DefaultHAL) DiskHealth() ([]DiskStatus, error) {
	devices := []string{"/dev/sda", "/dev/sdb", "/dev/nvme0n1"}
	var result []DiskStatus

	for _, dev := range devices {
		out, err := exec.Command("smartctl", "-H", dev).CombinedOutput()
		if err != nil {
			// ignore missing devices
			continue
		}
		raw := string(out)
		health := "UNKNOWN"
		if strings.Contains(raw, "PASSED") {
			health = "PASSED"
		} else if strings.Contains(raw, "FAILED") {
			health = "FAILED"
		}
		result = append(result, DiskStatus{
			Device: dev,
			Health: health,
			Raw:    raw,
		})
	}

	return result, nil
}
