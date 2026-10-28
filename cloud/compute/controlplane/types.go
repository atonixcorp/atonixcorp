package controlplane

import "time"

type ComputeType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Region struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Country        string `json:"country"`
	Available      bool   `json:"available"`
	CapacityStatus string `json:"capacity_status"`
}

type Image struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Family    string `json:"family"`
	Version   string `json:"version"`
	Source    string `json:"source"`
	Available bool   `json:"available"`
}

type Plan struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Family       string  `json:"family"`
	ComputeType  string  `json:"compute_type"`
	Region       string  `json:"region"`
	VCPUs        int     `json:"vcpus"`
	MemoryMB     int     `json:"memory_mb"`
	GPUCount     int     `json:"gpu_count"`
	BootDiskGB   int     `json:"boot_disk_gb"`
	TransferTB   int     `json:"transfer_tb"`
	HourlyPrice  float64 `json:"hourly_price"`
	MonthlyPrice float64 `json:"monthly_price"`
	Currency     string  `json:"currency"`
	Available    bool    `json:"available"`
}

type Network struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Region    string `json:"region"`
	Available bool   `json:"available"`
}

type SSHKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	KeyType     string `json:"key_type"`
	Available   bool   `json:"available"`
}

type Firewall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

type VolumeRequest struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	SizeGB int    `json:"size_gb"`
}

type InstanceRequest struct {
	Name        string          `json:"name"`
	Hostname    string          `json:"hostname"`
	Label       string          `json:"label"`
	Region      string          `json:"region"`
	ComputeType string          `json:"compute_type"`
	PlanID      string          `json:"plan_id"`
	ImageID     string          `json:"image_id"`
	AuthMethod  string          `json:"authentication_method"`
	NetworkID   string          `json:"network_id"`
	SSHKeyIDs   []string        `json:"ssh_key_ids"`
	FirewallID  string          `json:"firewall_id"`
	PublicIPv4  bool            `json:"public_ipv4"`
	PublicIPv6  bool            `json:"public_ipv6"`
	PrivateNet  bool            `json:"private_networking"`
	HostnameDNS bool            `json:"hostname_dns"`
	UserData    string          `json:"user_data"`
	Tags        []string        `json:"tags"`
	Quantity    int             `json:"quantity"`
	Volumes     []VolumeRequest `json:"volumes"`
}

type EstimateRequest struct {
	Region      string          `json:"region"`
	ComputeType string          `json:"compute_type"`
	PlanID      string          `json:"plan_id"`
	Quantity    int             `json:"quantity"`
	Volumes     []VolumeRequest `json:"volumes"`
}

type Estimate struct {
	HourlyPrice float64 `json:"hourly_price"`
	MonthlyPrice float64 `json:"monthly_price"`
	Quantity int `json:"quantity"`
	Currency string `json:"currency"`
	PricingSource string `json:"pricing_source"`
	Authoritative bool `json:"authoritative"`
}

type ProvisioningStep struct {
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	Message   string     `json:"message,omitempty"`
}

type Instance struct {
	ID           string             `json:"id"`
	DeploymentID string             `json:"deployment_id"`
	Name         string             `json:"name"`
	Hostname     string             `json:"hostname"`
	Label        string             `json:"label"`
	Region       string             `json:"region"`
	ComputeType  string             `json:"compute_type"`
	PlanID       string             `json:"plan_id"`
	ImageID      string             `json:"image_id"`
	AuthMethod   string             `json:"authentication_method"`
	NetworkID    string             `json:"network_id,omitempty"`
	SSHKeyIDs    []string           `json:"ssh_key_ids"`
	FirewallID   string             `json:"firewall_id,omitempty"`
	PublicIPv4   bool               `json:"public_ipv4"`
	PublicIPv6   bool               `json:"public_ipv6"`
	PrivateNet   bool               `json:"private_networking"`
	HostnameDNS  bool               `json:"hostname_dns"`
	UserData     string             `json:"user_data,omitempty"`
	Tags         []string           `json:"tags"`
	Volumes      []VolumeRequest    `json:"volumes"`
	Status       string             `json:"status"`
	Provisioning []ProvisioningStep `json:"provisioning"`
	HourlyPrice  float64            `json:"hourly_price"`
	MonthlyPrice float64            `json:"monthly_price"`
	Currency     string             `json:"currency"`
	NodeID       string             `json:"node_id,omitempty"`
	PublicIP     string             `json:"public_ip,omitempty"`
	PrivateIP    string             `json:"private_ip,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

type Catalog struct {
	ComputeTypes []ComputeType `json:"compute_types"`
	Regions      []Region      `json:"regions"`
	Images       []Image       `json:"images"`
	Plans        []Plan        `json:"plans"`
	Networks     []Network     `json:"networks"`
	SSHKeys      []SSHKey      `json:"ssh_keys"`
	Firewalls    []Firewall    `json:"firewalls"`
	VolumeMonthlyPricePerGB float64 `json:"volume_monthly_price_per_gb"`
}

func DefaultCatalog() Catalog {
	return Catalog{
		ComputeTypes: []ComputeType{
			{ID: "shared", Name: "Shared CPU", Description: "General workloads and development"},
			{ID: "performance", Name: "Performance Compute", Description: "Higher CPU and memory performance"},
			{ID: "dedicated", Name: "Dedicated CPU", Description: "Dedicated compute resources"},
			{ID: "gpu", Name: "GPU Compute", Description: "Accelerated AI and machine-learning workloads"},
		},
		Regions: []Region{
			{ID: "jhb-1", Name: "Johannesburg", Country: "South Africa", Available: true, CapacityStatus: "available"},
			{ID: "cpt-1", Name: "Cape Town", Country: "South Africa", Available: false, CapacityStatus: "unavailable"},
			{ID: "lon-1", Name: "London", Country: "United Kingdom", Available: false, CapacityStatus: "unavailable"},
			{ID: "fra-1", Name: "Frankfurt", Country: "Germany", Available: false, CapacityStatus: "unavailable"},
		},
		Images: []Image{
			{ID: "ubuntu-24-04", Name: "Ubuntu 24.04 LTS", Family: "Ubuntu", Version: "24.04 LTS", Source: "operating_system", Available: true},
			{ID: "ubuntu-22-04", Name: "Ubuntu 22.04 LTS", Family: "Ubuntu", Version: "22.04 LTS", Source: "operating_system", Available: true},
			{ID: "debian-12", Name: "Debian 12", Family: "Debian", Version: "12", Source: "operating_system", Available: true},
			{ID: "rocky-9", Name: "Rocky Linux 9", Family: "Rocky Linux", Version: "9", Source: "operating_system", Available: true},
		},
		Plans: []Plan{
			{ID: "gp-1c-1g", Name: "General Purpose 1", Family: "general", ComputeType: "shared", Region: "jhb-1", VCPUs: 1, MemoryMB: 1024, BootDiskGB: 25, TransferTB: 1, HourlyPrice: 0.02, MonthlyPrice: 14.60, Currency: "ZAR", Available: true},
			{ID: "gp-1c-2g", Name: "General Purpose 2", Family: "general", ComputeType: "shared", Region: "jhb-1", VCPUs: 1, MemoryMB: 2048, BootDiskGB: 50, TransferTB: 2, HourlyPrice: 0.04, MonthlyPrice: 29.20, Currency: "ZAR", Available: true},
			{ID: "gp-2c-4g", Name: "General Purpose 4", Family: "general", ComputeType: "shared", Region: "jhb-1", VCPUs: 2, MemoryMB: 4096, BootDiskGB: 80, TransferTB: 3, HourlyPrice: 0.08, MonthlyPrice: 58.40, Currency: "ZAR", Available: true},
			{ID: "gp-4c-8g", Name: "General Purpose 8", Family: "general", ComputeType: "shared", Region: "jhb-1", VCPUs: 4, MemoryMB: 8192, BootDiskGB: 160, TransferTB: 5, HourlyPrice: 0.16, MonthlyPrice: 116.80, Currency: "ZAR", Available: true},
			{ID: "cpu-4c-8g", Name: "CPU Optimized 8", Family: "cpu_optimized", ComputeType: "performance", Region: "jhb-1", VCPUs: 4, MemoryMB: 8192, BootDiskGB: 160, TransferTB: 5, HourlyPrice: 0.24, MonthlyPrice: 175.20, Currency: "ZAR", Available: true},
			{ID: "mem-2c-16g", Name: "Memory Optimized 16", Family: "memory_optimized", ComputeType: "performance", Region: "jhb-1", VCPUs: 2, MemoryMB: 16384, BootDiskGB: 160, TransferTB: 5, HourlyPrice: 0.24, MonthlyPrice: 175.20, Currency: "ZAR", Available: true},
			{ID: "io-4c-8g", Name: "Storage Optimized 8", Family: "storage_optimized", ComputeType: "performance", Region: "jhb-1", VCPUs: 4, MemoryMB: 8192, BootDiskGB: 320, TransferTB: 5, HourlyPrice: 0.28, MonthlyPrice: 204.40, Currency: "ZAR", Available: true},
			{ID: "dedicated-4c-8g", Name: "Dedicated CPU 8", Family: "general", ComputeType: "dedicated", Region: "jhb-1", VCPUs: 4, MemoryMB: 8192, BootDiskGB: 160, TransferTB: 5, HourlyPrice: 0.32, MonthlyPrice: 233.60, Currency: "ZAR", Available: true},
			{ID: "gpu-8c-32g-1", Name: "GPU Compute 32", Family: "gpu", ComputeType: "gpu", Region: "jhb-1", VCPUs: 8, MemoryMB: 32768, GPUCount: 1, BootDiskGB: 320, TransferTB: 8, HourlyPrice: 2.40, MonthlyPrice: 1752.00, Currency: "ZAR", Available: true},
		},
		Networks: []Network{
			{ID: "vpc-default-jhb-1", Name: "Default VPC", Region: "jhb-1", Available: true},
		},
		SSHKeys:   []SSHKey{},
		Firewalls: []Firewall{},
		VolumeMonthlyPricePerGB: 0.08,
	}
}
