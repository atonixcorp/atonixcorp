package api

import (
	"net/http"
	"time"

	"atonixcorp/cloud/hpc/api/handlers"
	"atonixcorp/cloud/hpc/api/middleware"
	"atonixcorp/cloud/hpc/jobs"
	"atonixcorp/cloud/hpc/metrics"
	"atonixcorp/cloud/hpc/nodes"
	"atonixcorp/cloud/hpc/scheduler"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// Simple backend selector: choose Slurm or K8s based on partition or config.
type BackendSelector = jobs.ProfileSelector

func NewBackendSelector(sl scheduler.Scheduler, ka scheduler.Scheduler) *BackendSelector {
	return jobs.NewProfileSelector(map[string]jobs.ExecutionTarget{
		"batch":     {Scheduler: sl, Backend: "slurm"},
		"container": {Scheduler: ka, Backend: "k8s"},
	})
}

func NewRouter(
	jobRepo jobs.Repository,
	eventPublisher jobs.EventPublisher,
	nodeInventory *nodes.InventoryService,
	logRepo jobs.Repository, // same repo if logs stored together
	exporter *metrics.Exporter,
	signingKey []byte,
	selector jobs.BackendSelector,
) http.Handler {
	r := chi.NewRouter()

	// Base chi middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))

	// Security & multi-tenant middleware
	auth := middleware.NewAuthMiddleware(signingKey)
	rate := middleware.NewRateLimiter(200)
	tenantScope := middleware.NewTenantScopeMiddleware()

	r.Use(auth.Middleware)
	r.Use(rate.Middleware)
	r.Use(tenantScope.Middleware)

	// Core services
	jobService := jobs.NewService(jobRepo, eventPublisher, selector)
	logService := jobs.NewLogService(logRepo)

	// Handlers
	jobSubmitHandler := handlers.NewJobSubmitHandler(jobService)
	jobStatusHandler := handlers.NewJobStatusHandler(jobService)
	jobLogsHandler := handlers.NewJobLogsHandler(logService)
	nodeListHandler := handlers.NewNodeListHandler(nodeInventory)
	partitionListHandler := handlers.NewPartitionListHandler(nodeInventory)
	metricsHandler := handlers.NewMetricsHandler(exporter)

	// Routes
	r.Route("/hpc", func(h chi.Router) {

		// Jobs
		h.Post("/jobs", jobSubmitHandler.ServeHTTP)
		h.Get("/jobs/{id}", jobStatusHandler.ServeHTTP)
		h.Get("/jobs/{id}/logs", jobLogsHandler.ServeHTTP)

		// Nodes & partitions
		h.Get("/nodes", nodeListHandler.ServeHTTP)
		h.Get("/partitions", partitionListHandler.ServeHTTP)
	})

	// Metrics (Prometheus)
	r.Handle("/metrics", metricsHandler.ServeHTTP)

	// Health
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return r
}
