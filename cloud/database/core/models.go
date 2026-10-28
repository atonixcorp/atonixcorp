package core

import "time"

// Supported engines
const (
	EnginePostgres  = "postgres"
	EngineTimescale = "timescale"
	EngineMongo     = "mongo"
	EngineRedis     = "redis"
	EngineMySQL     = "mysql"
	EngineMariaDB   = "mariadb"
)

type Plan struct {
	CPU          int         `yaml:"cpu"`
	MemoryMB     int         `yaml:"memory_mb"`
	StorageGB    int         `yaml:"storage_gb"`
	FlavorID     string      `yaml:"flavor_id"`
	HA           bool        `yaml:"ha"`
	Backups      string      `yaml:"backups"`
	ReadReplicas int         `yaml:"read_replicas"`
	Engines      []string    `yaml:"engines"`
	PriceUSD     float64     `yaml:"price_usd"`
	NodePricing  NodePricing `yaml:"node_pricing"`
}

type EngineCatalogEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Port        int      `json:"port"`
	Versions    []string `json:"versions"`
	Enabled     bool     `json:"enabled"`
}

type PlanCatalogEntry struct {
	ID        string   `json:"id"`
	CPU       int      `json:"cpu"`
	MemoryMB  int      `json:"memory_mb"`
	StorageGB int      `json:"storage_gb"`
	Engines   []string `json:"engines"`
	HighAvail bool     `json:"high_availability"`
	Enabled   bool     `json:"enabled"`
}

type NodePricing struct {
	CPUUSD     float64 `yaml:"cpu_usd"`
	MemoryUSD  float64 `yaml:"memory_usd"`
	StorageUSD float64 `yaml:"storage_usd"`
}

// DBInstance represents a database instance provisioned on OpenStack.
type DBInstance struct {
	ID              string `json:"id"`
	TenantID        string `json:"tenant_id"`
	ProjectID       string `json:"project_id"`
	Name            string `json:"name"`
	Engine          string `json:"engine"`
	EngineVersion   string `json:"engine_version"`
	Plan            string `json:"plan"`
	WorkloadProfile string `json:"workload_profile"`
	CPU             int    `json:"cpu"`
	MemoryMB        int    `json:"memory_mb"`
	StorageGB       int    `json:"storage_gb"`
	Region          string `json:"region"`
	Status          string `json:"status"`
	Endpoint        string `json:"endpoint"`
	Port            int    `json:"port"`
	NetworkID       string `json:"-"`

	// OpenStack resources
	OpenStackServerID       string `json:"-"`
	OpenStackVolumeID       string `json:"-"`
	OpenStackVolumeAttached bool   `json:"-"`
	OpenStackNetworkID      string `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DatabaseOperation struct {
	ID         string    `json:"id"`
	DatabaseID string    `json:"database_id"`
	TenantID   string    `json:"tenant_id"`
	ProjectID  string    `json:"project_id"`
	Status     string    `json:"status"`
	Stage      string    `json:"stage"`
	Reason     string    `json:"reason,omitempty"`
	Attempt    int       `json:"attempt"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateDatabaseResult struct {
	Database       *DBInstance        `json:"database"`
	Operation      *DatabaseOperation `json:"operation"`
	StatusURL      string             `json:"status_url"`
	EventStreamURL string             `json:"event_stream_url"`
}

// Backup metadata
type Backup struct {
	ID         string    `json:"id"`
	InstanceID string    `json:"instance_id"`
	Type       string    `json:"type"`     // full | snapshot
	Location   string    `json:"location"` // s3://...
	CreatedAt  time.Time `json:"created_at"`
}

type DBMetrics struct {
	InstanceID   string    `json:"instance_id"`
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  float64   `json:"memory_usage"`
	StorageUsage float64   `json:"storage_usage"`
	Connections  int       `json:"connections"`
	Timestamp    time.Time `json:"timestamp"`
}

// CreateInstanceRequest is the payload from API/UI.
type CreateInstanceRequest struct {
	TenantID        string `json:"tenant_id"`
	ProjectID       string `json:"project_id"`
	Name            string `json:"name"`
	Engine          string `json:"engine"`
	EngineVersion   string `json:"engine_version"`
	Plan            string `json:"plan"`
	WorkloadProfile string `json:"workload_profile"`
	CPU             int    `json:"cpu"`
	MemoryMB        int    `json:"memory_mb"`
	StorageGB       int    `json:"storage_gb"`
	Region          string `json:"region"`
}
