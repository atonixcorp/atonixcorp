package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"atonixcorp/cloud/hpc/api/handlers"
	"atonixcorp/cloud/hpc/api/middleware"
	"atonixcorp/cloud/hpc/datasets"
	"atonixcorp/cloud/hpc/jobs"
	"atonixcorp/cloud/hpc/metrics"
	"atonixcorp/cloud/hpc/nodes"
	"atonixcorp/cloud/hpc/scheduler"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// Simple backend selector: choose Slurm or K8s based on partition or config.
type BackendSelector = jobs.ProfileSelector

type RuntimeChecks struct {
	PingDatabase    func(context.Context) error
	CheckStorage     func(context.Context) error
	SlurmConfigured bool
	CheckSlurm      func(context.Context) error
	DatasetMetadata datasets.MetadataRepository
	DatasetUpload   *datasets.UploadService
	DatasetDownload *datasets.DownloadService
}

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
	return NewRouterWithChecks(jobRepo, eventPublisher, nodeInventory, logRepo, exporter,
		signingKey, selector, RuntimeChecks{})
}

func NewRouterWithChecks(
	jobRepo jobs.Repository,
	eventPublisher jobs.EventPublisher,
	nodeInventory *nodes.InventoryService,
	logRepo jobs.Repository,
	exporter *metrics.Exporter,
	signingKey []byte,
	selector jobs.BackendSelector,
	checks RuntimeChecks,
) http.Handler {
	r := chi.NewRouter()

	// Base chi middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))

	auth := middleware.NewAuthMiddleware(signingKey)
	rate := middleware.NewRateLimiter(200)
	tenantScope := middleware.NewTenantScopeMiddleware()

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/ready", func(w http.ResponseWriter, request *http.Request) {
		if checks.PingDatabase != nil {
			if err := checks.PingDatabase(request.Context()); err != nil {
				log.Printf("HPC readiness check failed: PostgreSQL: %v", err)
				http.Error(w, "database unavailable", http.StatusServiceUnavailable)
				return
			}
		}
		if checks.CheckStorage != nil {
			if err := checks.CheckStorage(request.Context()); err != nil {
				log.Printf("HPC readiness check failed: object storage: %v", err)
				http.Error(w, "object storage unavailable", http.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	r.Get("/integrations/slurm", func(w http.ResponseWriter, request *http.Request) {
		result := map[string]any{"configured": checks.SlurmConfigured, "available": false}
		if !checks.SlurmConfigured {
			result["status"] = "disabled"
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(result)
			return
		}
		if checks.CheckSlurm == nil {
			result["status"] = "unavailable"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(result)
			return
		}
		if err := checks.CheckSlurm(request.Context()); err != nil {
			log.Printf("Slurm availability check failed: %v", err)
			result["status"] = "unavailable"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(result)
			return
		}
		result["available"] = true
		result["status"] = "available"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})

	protected := chi.NewRouter()
	protected.Use(auth.Middleware)
	protected.Use(rate.Middleware)
	protected.Use(tenantScope.Middleware)

	jobService := jobs.NewService(jobRepo, eventPublisher, selector)
	logService := jobs.NewLogService(logRepo)
	jobSubmitHandler := handlers.NewJobSubmitHandler(jobService)
	jobStatusHandler := handlers.NewJobStatusHandler(jobService)
	jobLogsHandler := handlers.NewJobLogsHandler(logService)
	nodeListHandler := handlers.NewNodeListHandler(nodeInventory)
	partitionListHandler := handlers.NewPartitionListHandler(nodeInventory)
	metricsHandler := handlers.NewMetricsHandler(exporter)

	protected.Route("/hpc", func(h chi.Router) {
		h.Post("/jobs", jobSubmitHandler.ServeHTTP)
		h.Get("/jobs/{id}", jobStatusHandler.ServeHTTP)
		h.Get("/jobs/{id}/logs", jobLogsHandler.ServeHTTP)
		h.Get("/nodes", nodeListHandler.ServeHTTP)
		h.Get("/partitions", partitionListHandler.ServeHTTP)
		if checks.DatasetMetadata != nil && checks.DatasetUpload != nil && checks.DatasetDownload != nil {
			datasetHandlers := &handlers.DatasetHandlers{
				Metadata: checks.DatasetMetadata,
				Upload:   checks.DatasetUpload,
				Download: checks.DatasetDownload,
			}
			h.Get("/datasets", datasetHandlers.List)
			h.Post("/datasets", datasetHandlers.UploadDataset)
			h.Get("/datasets/{id}/download", datasetHandlers.DownloadDataset)
		}
	})
	protected.Handle("/metrics", metricsHandler)
	r.Mount("/", protected)
	return r
}
