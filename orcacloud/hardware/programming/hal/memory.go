package hal

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// MemoryUsage returns memory usage percentage.
func (h *DefaultHAL) MemoryUsage() (float64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var total, available float64

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseFloat(fields[1], 64)
				total = v
			}
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseFloat(fields[1], 64)
				available = v
			}
		}
	}
	if total == 0 {
		return 0, nil
	}

	used := total - available
	usage := (used / total) * 100.0
	return usage, nil
}
