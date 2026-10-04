package nvme

import "os/exec"

// ResetController performs a controller-level reset.
func ResetController(dev Device) error {
	_, err := exec.Command("nvme", "reset", dev.Path).CombinedOutput()
	return err
}

// FormatDevice formats the NVMe namespace.
func FormatDevice(dev Device) error {
	_, err := exec.Command("nvme", "format", dev.Path).CombinedOutput()
	return err
}

// ListNamespaces returns namespace info.
func ListNamespaces(dev Device) (string, error) {
	out, err := exec.Command("nvme", "list-ns", dev.Path).CombinedOutput()
	return string(out), err
}
