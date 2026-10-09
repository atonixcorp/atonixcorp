package core

// -----------------------------
// NETWORK
// -----------------------------
type Network struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	CIDR     string `json:"cidr"`
	TenantID string `json:"tenant_id"`
	External bool   `json:"external"`
}

// -----------------------------
// SUBNET
// -----------------------------
type Subnet struct {
	ID        string `json:"id"`
	NetworkID string `json:"network_id"`
	CIDR      string `json:"cidr"`
	GatewayIP string `json:"gateway_ip"`
}

// -----------------------------
// ROUTER
// -----------------------------
type Router struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	TenantID          string `json:"tenant_id"`
	ExternalNetworkID string `json:"external_network_id"`
}

// -----------------------------
// ROUTER INTERFACE
// -----------------------------
type RouterInterface struct {
	ID        string `json:"id"`
	RouterID  string `json:"router_id"`
	SubnetID  string `json:"subnet_id"`
	IPAddress string `json:"ip_address"`
}

// -----------------------------
// PORT
// -----------------------------
type Port struct {
	ID        string `json:"id"`
	NetworkID string `json:"network_id"`
	DeviceID  string `json:"device_id"`
	MAC       string `json:"mac"`
}

// -----------------------------
// FLOATING IP
// -----------------------------
type FloatingIP struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	ExternalIP     string `json:"external_ip"`
	InternalPortID string `json:"internal_port_id"`
	InternalIP     string `json:"internal_ip"`
}

// -----------------------------
// NAT RULE
// -----------------------------
type NATRule struct {
	ID         string `json:"id"`
	RouterID   string `json:"router_id"`
	ExternalIP string `json:"external_ip"`
	InternalIP string `json:"internal_ip"`
	Type       string `json:"type"` // snat | dnat
}

// -----------------------------
// SECURITY GROUP + RULES
// -----------------------------
type SecurityRule struct {
	ID          string `json:"id"`
	Direction   string `json:"direction"`  // ingress | egress
	Protocol    string `json:"protocol"`   // tcp | udp | icmp | any
	PortRange   string `json:"port_range"` // "80", "443", "1024-65535", "*"
	CIDR        string `json:"cidr"`       // e.g. "0.0.0.0/0"
	Description string `json:"description"`
}

type SecurityGroup struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	TenantID string         `json:"tenant_id"`
	Rules    []SecurityRule `json:"rules"`
}
