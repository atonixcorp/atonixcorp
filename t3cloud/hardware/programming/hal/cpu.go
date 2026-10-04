package hal

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// CPUUsage returns an approximate CPU usage percentage.
// This is a simple implementation; you can replace with gopsutil later.
func (h *DefaultHAL) CPUUsage() (float64, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return 0, scanner.Err()
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, nil
	}

	var nums []float64
	for _, f := range fields[1:] {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return 0, err
		}
		nums = append(nums, v)
	}

	if len(nums) < 4 {
		return 0, nil
	}

	idle := nums[3]
	total := 0.0
	for _, v := range nums {
		total += v
	}
	if total == 0 {
		return 0, nil
	}

	usage := (1.0 - idle/total) * 100.0
	return usage, nil
}
