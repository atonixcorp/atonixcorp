package nvme

import (
	"os/exec"
)

// Benchmark runs a simple fio test on the NVMe device.
func Benchmark(dev Device) (string, error) {
	cmd := exec.Command("fio",
		"--name=nvmebench",
		"--filename="+dev.Path,
		"--rw=randread",
		"--bs=4k",
		"--size=1G",
		"--numjobs=1",
		"--iodepth=32",
		"--time_based",
		"--runtime=10",
		"--group_reporting",
	)

	out, err := cmd.CombinedOutput()
	return string(out), err
}
