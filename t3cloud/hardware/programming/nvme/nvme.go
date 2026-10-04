package nvme

import (
	"os/exec"
	"strings"
)

// Device represents an NVMe device.
type Device struct {
	Path string // e.g. "/dev/nvme0n1"
}

// DiscoverDevices scans the system for NVMe devices using `nvme list`.
func DiscoverDevices() ([]Device, error) {
	out, err := exec.Command("nvme", "list").CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var devices []Device

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "/dev/nvme") {
			fields := strings.Fields(line)
			dev := fields[0] // first column is the device path
			devices = append(devices, Device{Path: dev})
		}
	}

	return devices, nil
}
