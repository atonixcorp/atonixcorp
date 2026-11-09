package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"atonixcorp/cloud/hpc/api"
	"atonixcorp/cloud/hpc/config"
	"atonixcorp/cloud/hpc/datasets"
	"atonixcorp/cloud/hpc/jobs"
	"atonixcorp/cloud/hpc/metrics"
	"atonixcorp/cloud/hpc/nodes"
	"atonixcorp/cloud/hpc/openhpc/resource_managers/slurm"
	"atonixcorp/cloud/hpc/repository"
	"atonixcorp/cloud/hpc/scheduler"
	"atonixcorp/cloud/hpc/scheduler/k8s"
)

func main() {
	if err := run(); err != nil {
		log.Printf("HPC service stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("config/hpc.yaml")
	if err != nil {
		return fmt.Errorf("load HPC configuration: %w", err)
	}
	discoveryInterval, err := time.ParseDuration(cfg.Cluster.DiscoveryInterval)
	if err != nil {
		return fmt.Errorf("parse node discovery interval: %w", err)
	}
	healthInterval, err := time.ParseDuration(cfg.Cluster.HealthInterval)
	if err != nil {
		return fmt.Errorf("parse node health interval: %w", err)
	}
	log.Printf("[hpc] starting service=%s", cfg.Service.Name)

	startupCtx, cancelStartup := context.WithTimeout(ctx, time.Minute)
	defer cancelStartup()
	repositories, err := repository.OpenPostgres(startupCtx, cfg.DatabaseDSN())
	if err != nil {
		return err
	}
	defer func() {
		if err := repositories.Close(); err != nil {
			log.Printf("close HPC database connection: %v", err)
		}
	}()
	if err := repositories.Migrate(startupCtx); err != nil {
		return fmt.Errorf("initialize HPC persistence: %w", err)
	}

	storage, err := datasets.NewS3Adapter(
		cfg.Storage.S3.Endpoint,
		cfg.Storage.S3.AccessKey,
		cfg.Storage.S3.SecretKey,
		cfg.Storage.S3.Bucket,
		cfg.Storage.S3.Secure,
	)
	if err != nil {
		return fmt.Errorf("initialize dataset storage: %w", err)
	}
	if err := ensureDatasetBucket(startupCtx, storage); err != nil {
		return err
	}

	var slurmClient *slurm.SlurmClient
	var slurmAdapter scheduler.Scheduler
	if cfg.Scheduler.Slurm.Enabled {
		timeout, err := time.ParseDuration(cfg.Scheduler.Slurm.CommandTimeout)
		if err != nil {
			return fmt.Errorf("parse Slurm command timeout: %w", err)
		}
		slurmClient, err = slurm.NewSlurmClientWithConfig(slurm.ClientConfig{
			SbatchPath:   cfg.Scheduler.Slurm.SbatchPath,
			SqueuePath:   cfg.Scheduler.Slurm.SqueuePath,
			SacctPath:    cfg.Scheduler.Slurm.SacctPath,
			ScancelPath:  cfg.Scheduler.Slurm.ScancelPath,
			ScontrolPath: cfg.Scheduler.Slurm.ScontrolPath,
			Timeout:      timeout,
		})
		if err != nil {
			return fmt.Errorf("configure Slurm integration: %w", err)
		}
		slurmAdapter = slurm.NewSlurmAdapterWithClient(slurmClient)
		if err := slurmClient.CheckAvailability(startupCtx); err != nil {
			log.Printf("[hpc] Slurm is configured but unavailable: %v", err)
		}
	}

	var k8sAdapter scheduler.Scheduler
	if cfg.Scheduler.K8s.Enabled {
		k8sAdapter, err = k8s.NewK8sAdapter(cfg.Scheduler.K8s.Namespace)
		if err != nil {
			return fmt.Errorf("initialize Kubernetes scheduler: %w", err)
		}
	}
	selector, err := buildSelector(cfg, slurmAdapter, k8sAdapter)
	if err != nil {
		return err
	}

	inventory := nodes.NewInventoryService(repositories)
	healthProbe, err := nodes.NewTCPHealthProbe(cfg.Cluster.HealthPort, 3*time.Second)
	if err != nil {
		return fmt.Errorf("configure node health probe: %w", err)
	}
	health := nodes.NewHealthService(inventory, healthProbe)

	if slurmClient != nil {
		discovery := nodes.NewDiscoveryService(inventory, slurmClient)
		discoveryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		if err := discovery.SyncClusterNodes(discoveryCtx); err != nil {
			log.Printf("[hpc] initial Slurm discovery failed: %v", err)
		}
		cancel()
		discovery.StartPeriodicSync(ctx, discoveryInterval)
	}
	go startHealthLoop(ctx, health, healthInterval)

	exporter := metrics.NewExporter()
	exporterErrors := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Metrics.Port)
		log.Printf("[hpc] metrics listening on %s", addr)
		if err := exporter.ServeContext(ctx, addr); err != nil {
			exporterErrors <- fmt.Errorf("serve HPC metrics: %w", err)
		}
	}()

	router := api.NewRouterWithChecks(
		repositories,
		repositories,
		inventory,
		repositories,
		exporter,
		[]byte(cfg.Auth.JWTSigningKey),
		selector,
		api.RuntimeChecks{
			PingDatabase:    repositories.Ping,
			CheckStorage:    storage.CheckBucket,
			SlurmConfigured: slurmClient != nil,
			CheckSlurm: func(checkCtx context.Context) error {
				if slurmClient == nil {
					return errors.New("Slurm is disabled")
				}
				return slurmClient.CheckAvailability(checkCtx)
			},
			DatasetMetadata: repositories,
			DatasetUpload:   datasets.NewUploadService(storage, repositories),
			DatasetDownload: datasets.NewDownloadService(storage, repositories),
		},
	)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Service.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("[hpc] API listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("serve HPC API: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErrors:
		return err
	case err := <-exporterErrors:
		return err
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down HPC API: %w", err)
	}
	return nil
}

func buildSelector(cfg *config.HPCConfig, slurmBackend, k8sBackend scheduler.Scheduler) (jobs.BackendSelector, error) {
	targets := make(map[string]jobs.ExecutionTarget)
	if len(cfg.Scheduler.ExecutionProfiles) > 0 {
		for name, profile := range cfg.Scheduler.ExecutionProfiles {
			var backend scheduler.Scheduler
			switch profile.Backend {
			case "slurm":
				backend = slurmBackend
			case "k8s":
				backend = k8sBackend
			default:
				return nil, fmt.Errorf("execution profile %q has unsupported backend %q", name, profile.Backend)
			}
			if backend == nil {
				return nil, fmt.Errorf("execution profile %q requires disabled backend %q", name, profile.Backend)
			}
			targets[name] = jobs.ExecutionTarget{
				Scheduler: backend,
				Backend:   profile.Backend,
				Partition: profile.Partition,
				Account:   profile.Account,
			}
		}
		return jobs.NewProfileSelector(targets), nil
	}

	switch cfg.Scheduler.DefaultBackend {
	case "slurm":
		if slurmBackend == nil {
			return nil, errors.New("default scheduler is Slurm but Slurm is disabled")
		}
		targets["batch"] = jobs.ExecutionTarget{Scheduler: slurmBackend, Backend: "slurm"}
	case "k8s":
		if k8sBackend == nil {
			return nil, errors.New("default scheduler is Kubernetes but Kubernetes is disabled")
		}
		targets["container"] = jobs.ExecutionTarget{Scheduler: k8sBackend, Backend: "k8s"}
	}
	return jobs.NewProfileSelector(targets), nil
}

func ensureDatasetBucket(ctx context.Context, storage *datasets.S3Adapter) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := storage.EnsureBucket(ctx); err == nil {
			return nil
		} else if ctx.Err() != nil {
			return fmt.Errorf("ensure dataset bucket before startup: %w", err)
		} else {
			log.Printf("[hpc] waiting for dataset storage: %v", err)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("ensure dataset bucket before startup: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func startHealthLoop(ctx context.Context, service *nodes.HealthService, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			checkCtx, cancel := context.WithTimeout(ctx, interval)
			if err := service.RefreshAllNodeHealth(checkCtx); err != nil {
				log.Printf("[hpc] node health refresh failed: %v", err)
			}
			cancel()
		case <-ctx.Done():
			return
		}
	}
}
