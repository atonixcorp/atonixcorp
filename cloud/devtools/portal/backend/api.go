package backend

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"atonixcorp/cloud/devtools/portal/backend/auth"
	"atonixcorp/cloud/devtools/portal/backend/handlers"
)

func Start(addr string) error {
	if os.Getenv("ATCLOUD_PORTAL_TOKEN") == "" {
		return fmt.Errorf("ATCLOUD_PORTAL_TOKEN is required")
	}
	if _, err := handlers.ValidateUpstreamConfiguration(); err != nil {
		return err
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handlers.Health)
	mux.HandleFunc("GET /ready", handlers.Ready)
	mux.HandleFunc("GET /version", handlers.Version)

	registerProtected := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, auth.WithAuth(handler))
	}
	registerProtected("GET /projects", handlers.ListProjects)
	registerProtected("GET /compute", handlers.ListCompute)
	registerProtected("POST /compute/create", handlers.CreateCompute)
	registerProtected("DELETE /compute/delete/{id}", handlers.DeleteCompute)
	registerProtected("GET /network", handlers.ListNetwork)
	registerProtected("POST /network/create", handlers.CreateNetwork)
	registerProtected("DELETE /network/delete/{id}", handlers.DeleteNetwork)
	registerProtected("GET /storage", handlers.ListStorage)
	registerProtected("POST /storage/create", handlers.CreateStorage)
	registerProtected("DELETE /storage/delete/{id}", handlers.DeleteStorage)
	registerProtected("GET /telemetry", handlers.Telemetry)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("DevTools Portal backend listening on %s", addr)
	return srv.ListenAndServe()
}
