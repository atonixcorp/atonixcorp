package nvme

import (
	"os/exec"
	"strings"
)

type Health struct {
	Temperature    string
	AvailableSpare string
	MediaErrors    string
	PowerCycles    string
	Raw            string
}

// ReadHealth returns SMART/health info for an NVMe device.
func ReadHealth(dev Device) (Health, error) {
	out, err := exec.Command("nvme", "smart-log", dev.Path).CombinedOutput()
	if err != nil {
		return Health{}, err
	}

	raw := string(out)
	h := Health{Raw: raw}

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "temperature") {
			h.Temperature = line
		}
		if strings.Contains(line, "available_spare") {
			h.AvailableSpare = line
		}
		if strings.Contains(line, "media_errors") {
			h.MediaErrors = line
		}
		if strings.Contains(line, "power_cycles") {
			h.PowerCycles = line
		}
	}

	return h, nil
}
