package metrics

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Exporter struct {
	Collectors *HPCCollectors
	Registry   *prometheus.Registry
}

func NewExporter() *Exporter {
	reg := prometheus.NewRegistry()
	collectors := NewHPCCollectors()
	collectors.Register(reg)

	return &Exporter{
		Collectors: collectors,
		Registry:   reg,
	}
}

func (e *Exporter) Handler() http.Handler {
	return promhttp.HandlerFor(e.Registry, promhttp.HandlerOpts{})
}

func (e *Exporter) Serve(addr string) error {
	return e.ServeContext(context.Background(), addr)
}

func (e *Exporter) ServeContext(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", e.Handler())
	log.Printf("[metrics] exporter listening on %s", addr)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()
	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
