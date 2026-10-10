package nvme

import (
	"os/exec"
	"strings"
)

type Info struct {
	Model    string
	Serial   string
	Firmware string
	Raw      string
}

// ReadInfo returns model, serial, firmware for an NVMe device.
func ReadInfo(dev Device) (Info, error) {
	out, err := exec.Command("nvme", "id-ctrl", dev.Path).CombinedOutput()
	if err != nil {
		return Info{}, err
	}

	raw := string(out)
	info := Info{Raw: raw}

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "mn") {
			info.Model = line
		}
		if strings.HasPrefix(line, "sn") {
			info.Serial = line
		}
		if strings.HasPrefix(line, "fr") {
			info.Firmware = line
		}
	}

	return info, nil
}
