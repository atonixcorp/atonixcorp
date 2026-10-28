package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"atnetwork/agents/node"
	"atnetwork/api"
	"atnetwork/backends/atcloud"
	"atnetwork/backends/datacenter"
	"atnetwork/backends/generic"
	"atnetwork/backends/kubernetes"
	"atnetwork/backends/openstack"
	"atnetwork/config"
	"atnetwork/core"
)

func main() {
	configDir := os.Getenv("ATNETWORK_CONFIG_DIR")
	if configDir == "" {
		configDir = "./config"
	}
	envCfg, err := config.LoadEnv(configDir + "/env.yaml")
	if err != nil {
		log.Fatalf("load networking environment config: %v", err)
	}

	backendsCfg, err := config.LoadBackends(configDir + "/backends.yaml")
	if err != nil {
		log.Fatalf("load networking backend config: %v", err)
	}

	dsn := os.Getenv("ATNETWORK_DATABASE_DSN")
	if dsn == "" {
		dsn = envCfg.DatabaseConnectionString()
	}
	store, err := core.NewPostgresStore(dsn)
	if err != nil {
		log.Fatalf("connect networking store: %v", err)
	}
	defer store.Close()
	migrationsDir := os.Getenv("ATNETWORK_MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "./migrations"
	}
	if err := store.Migrate(migrationsDir); err != nil {
		log.Fatalf("migrate networking store: %v", err)
	}

	var backends []core.Backend

	if backendsCfg.OVN.Enabled {
		backends = append(backends, atcloud.NewOVNBackend())
	}

	if backendsCfg.FRR.Enabled {
		backends = append(backends, atcloud.NewFRRBackend(backendsCfg.FRR.ConfigPath))
	}

	if backendsCfg.Dnsmasq.Enabled {
		backends = append(backends, atcloud.NewDnsmasqBackend(backendsCfg.Dnsmasq.ConfigDir))
	}

	if backendsCfg.Datacenter.Enabled {
		backends = append(backends, datacenter.NewFabricBackend())
	}

	if backendsCfg.Kubernetes.Enabled {
		backends = append(backends, kubernetes.NewCNIBackend())
	}

	if backendsCfg.Neutron.Enabled {
		neutronClient := openstack.NewNeutronClient(
			backendsCfg.Neutron.Endpoint,
			backendsCfg.Neutron.Token,
		)
		backends = append(backends, openstack.NewNeutronBackend(neutronClient))
	}

	if backendsCfg.Generic.Enabled {
		backends = append(backends, generic.NewStaticBackend())
	}

	fmt.Println("ATN: Loaded backends:", len(backends))

	controller := core.NewController(store, backends)
	policy := core.NewPolicy()

	reconciler := core.NewReconciler(
		store,
		backends,
		time.Duration(envCfg.Reconciler.IntervalSeconds)*time.Second,
	)
	reconciler.Start()

	if envCfg.Agent.Enabled {
		agent := node.NewNodeAgent(envCfg.Environment.NodeID, store)
		agent.Start()
		fmt.Println("ATN: Node agent started for node:", envCfg.Environment.NodeID)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	fmt.Println("ATN: Networking API starting on :8080")
	if err := api.StartHTTP(controller, policy, store, jwtSecret); err != nil {
		log.Fatalf("start networking API: %v", err)
	}
}
