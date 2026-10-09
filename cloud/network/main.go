package atnetwork

import (
	"fmt"
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
	// ---------------------------------------------------------
	// 1. Load configuration
	// ---------------------------------------------------------
	envCfg, err := config.LoadEnv("atnetwork/config/env.yaml")
	if err != nil {
		panic(fmt.Errorf("failed to load env.yaml: %w", err))
	}

	backendsCfg, err := config.LoadBackends("atnetwork/config/backends.yaml")
	if err != nil {
		panic(fmt.Errorf("failed to load backends.yaml: %w", err))
	}

	// ---------------------------------------------------------
	// 2. Initialize PostgreSQL store
	// ---------------------------------------------------------
	store, err := core.NewPostgresStore(envCfg.DatabaseConnectionString())
	if err != nil {
		panic(fmt.Errorf("failed to connect to PostgreSQL: %w", err))
	}

	// ---------------------------------------------------------
	// 3. Initialize backends dynamically
	// ---------------------------------------------------------
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

	// ---------------------------------------------------------
	// 4. Initialize controller
	// ---------------------------------------------------------
	controller := core.NewController(store, backends)
	policy := core.NewPolicy()

	// ---------------------------------------------------------
	// 5. Start reconciler
	// ---------------------------------------------------------
	reconciler := core.NewReconciler(
		store,
		backends,
		time.Duration(envCfg.Reconciler.IntervalSeconds)*time.Second,
	)
	reconciler.Start()

	fmt.Println("ATN: Reconciler started")

	// ---------------------------------------------------------
	// 6. Start node agent (per-node)
	// ---------------------------------------------------------
	if envCfg.Agent.Enabled {
		agent := node.NewNodeAgent(envCfg.Environment.NodeID, store)
		agent.Start()
		fmt.Println("ATN: Node agent started for node:", envCfg.Environment.NodeID)
	}

	// ---------------------------------------------------------
	// 7. Start API server
	// ---------------------------------------------------------
	fmt.Println("ATN: API server starting on :8080")
	if err := api.StartHTTP(controller, policy); err != nil {
		panic(err)
	}
}
