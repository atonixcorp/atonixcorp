package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	// Core HPC modules
	"atonixcorp/cloud/hpc/api"
	"atonixcorp/cloud/hpc/config"
	"atonixcorp/cloud/hpc/datasets"
	"atonixcorp/cloud/hpc/jobs"
	"atonixcorp/cloud/hpc/metrics"
	"atonixcorp/cloud/hpc/nodes"
	"atonixcorp/cloud/hpc/openhpc/resource_managers/slurm"
	"atonixcorp/cloud/hpc/scheduler"
	"atonixcorp/cloud/hpc/scheduler/k8s"
)

func main() {
	// ------------------------------------------------------------
	// 1. Load configuration
	// ------------------------------------------------------------
	cfg, err := config.Load("config/hpc.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Printf("[hpc] starting service=%s", cfg.Service.Name)

	// ------------------------------------------------------------
	// 2. Initialize dataset storage (S3 / MinIO / Ceph)
	// ------------------------------------------------------------
	s3, err := datasets.NewS3Adapter(
		cfg.Storage.S3.Endpoint,
		cfg.Storage.S3.AccessKey,
		cfg.Storage.S3.SecretKey,
		cfg.Storage.S3.Bucket,
		cfg.Storage.S3.Secure,
	)
	if err != nil {
		log.Fatalf("failed to init S3 storage: %v", err)
	}

	// ------------------------------------------------------------
	// 3. Initialize repositories (PostgreSQL)
	// ------------------------------------------------------------
	// You will implement these repositories.
	var jobRepo jobs.Repository
	var metaRepo datasets.MetadataRepository
	var eventPublisher jobs.EventPublisher

	// ------------------------------------------------------------
	// 4. Initialize scheduler backends
	// ------------------------------------------------------------
	var slurmAdapter, k8sAdapter scheduler.Scheduler
	if cfg.Scheduler.Slurm.Enabled {
		slurmAdapter = slurm.NewSlurmAdapter()
	}
	if cfg.Scheduler.K8s.Enabled {
		k8sAdapter, err = k8s.NewK8sAdapter(cfg.Scheduler.K8s.Namespace)
		if err != nil {
			log.Fatalf("failed to init k8s adapter: %v", err)
		}
	}

	var selector jobs.BackendSelector = api.NewBackendSelector(slurmAdapter, k8sAdapter)
	if len(cfg.Scheduler.ExecutionProfiles) > 0 {
		targets := make(map[string]jobs.ExecutionTarget, len(cfg.Scheduler.ExecutionProfiles))
		for name, profile := range cfg.Scheduler.ExecutionProfiles {
			var backend scheduler.Scheduler
			switch profile.Backend {
			case "slurm":
				backend = slurmAdapter
			case "k8s":
				backend = k8sAdapter
			default:
				log.Fatalf("execution profile %s has an unknown backend", name)
			}
			if backend == nil {
				log.Fatalf("execution profile %s selects a disabled backend", name)
			}
			targets[name] = jobs.ExecutionTarget{Scheduler: backend, Backend: profile.Backend, Partition: profile.Partition, Account: profile.Account}
		}
		selector = jobs.NewProfileSelector(targets)
	}

	// ------------------------------------------------------------
	// 5. Initialize job services
	// ------------------------------------------------------------

	// ------------------------------------------------------------
	// 6. Initialize node inventory + discovery + health
	// ------------------------------------------------------------
	nodeInventory := nodes.NewInventoryService(nil) // plug in your repo

	// Discovery client (Slurm or K8s)
	var discoveryClient nodes.DiscoveryClient
	discoveryClient = nodes.NewSlurmDiscoveryClient() // implement this

	discovery := nodes.NewDiscoveryService(nodeInventory, discoveryClient)
	health := nodes.NewHealthService(nodeInventory, nil) // plug in prober

	// Start background loops
	go discovery.StartPeriodicSync(context.Background(), mustDuration(cfg.Cluster.DiscoveryInterval))
	go func() {
		ticker := time.NewTicker(mustDuration(cfg.Cluster.HealthInterval))
		for range ticker.C {
			_ = health.RefreshAllNodeHealth(context.Background())
		}
	}()

	// ------------------------------------------------------------
	// 7. Initialize metrics exporter
	// ------------------------------------------------------------
	exporter := metrics.NewExporter()
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Metrics.Port)
		log.Printf("[metrics] listening on %s", addr)
		if err := exporter.Serve(addr); err != nil {
			log.Fatalf("metrics server failed: %v", err)
		}
	}()

	// ------------------------------------------------------------
	// 8. Build router
	// ------------------------------------------------------------
	router := api.NewRouter(
		jobRepo,
		eventPublisher,
		nodeInventory,
		jobRepo,
		exporter,
		[]byte(cfg.Auth.JWTSigningKey),
		selector,
	)

	// ------------------------------------------------------------
	// 9. Start HTTP server
	// ------------------------------------------------------------
	addr := fmt.Sprintf(":%d", cfg.Service.Port)
	log.Printf("[hpc] API listening on %s", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func mustDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatalf("invalid duration: %s", s)
	}
	return d
}
