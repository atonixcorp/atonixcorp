package analytics

import "sync"

var mu sync.Mutex
var LocalMetrics = map[string]int{}

func Inc(metric string) {
	mu.Lock()
	LocalMetrics[metric]++
	mu.Unlock()
}

func FlushLocalMetrics() {
	payload := make(map[string]any, len(LocalMetrics))
	for k, v := range LocalMetrics {
		payload[k] = v
	}
	SendEvent("cli-metrics", payload)
}
