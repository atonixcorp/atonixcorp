package telemetry

import (
	"fmt"
	"net/http"
	"strings"
)

// Exporter exposes metrics to the outside world (e.g., Prometheus).
type Exporter struct {
	collector *Collector
}

// NewExporter creates a new exporter.
func NewExporter(c *Collector) *Exporter {
	return &Exporter{collector: c}
}

// Handler returns an http.Handler that serves metrics in a simple text format.
// You can later replace this with full Prometheus exposition.
func (e *Exporter) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics, err := e.collector.CollectMetrics()
		if err != nil {
			http.Error(w, "failed to collect metrics: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var b strings.Builder
		for _, m := range metrics {
			fmt.Fprintf(&b, "%s %f\n", m.Name, m.Value)
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(b.String()))
	})
}
