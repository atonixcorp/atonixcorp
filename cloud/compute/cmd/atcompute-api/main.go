package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	computeapi "atonixcorp/cloud/compute/api"
	"atonixcorp/cloud/compute/controlplane"
)

func main() {
	storePath := envOrDefault("COMPUTE_STORE_PATH", "/var/lib/atonixcorp-compute/instances.json")
	store, err := controlplane.OpenFileStore(storePath)
	if err != nil {
		log.Fatalf("open compute instance store: %v", err)
	}

	handler, err := computeapi.NewServer(
		controlplane.DefaultCatalog(),
		store,
		os.Getenv("COMPUTE_API_TOKEN"),
		os.Getenv("COMPUTE_NODE_TOKEN"),
	)
	if err != nil {
		log.Fatalf("configure compute API: %v", err)
	}

	server := &http.Server{
		Addr:              envOrDefault("COMPUTE_API_ADDR", ":8580"),
		Handler:           handler.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-shutdownContext.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("shutdown compute API: %v", err)
		}
	}()

	log.Printf("compute API listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve compute API: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
