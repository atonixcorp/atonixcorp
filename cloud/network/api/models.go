package api

type CreateNetworkRequest struct {
	Name     string `json:"name"`
	CIDR     string `json:"cidr"`
	TenantID string `json:"tenant_id"`
	External bool   `json:"external"`
}

type CreateSubnetRequest struct {
	NetworkID string `json:"network_id"`
	CIDR      string `json:"cidr"`
	GatewayIP string `json:"gateway_ip"`
}

type CreateRouterRequest struct {
	Name              string `json:"name"`
	TenantID          string `json:"tenant_id"`
	ExternalNetworkID string `json:"external_network_id"`
}

type AttachRouterInterfaceRequest struct {
	RouterID  string `json:"router_id"`
	SubnetID  string `json:"subnet_id"`
	IPAddress string `json:"ip_address"`
}

type CreatePortRequest struct {
	NetworkID string `json:"network_id"`
	DeviceID  string `json:"device_id"`
	MAC       string `json:"mac"`
}

type CreateSecurityGroupRequest struct {
	Name     string `json:"name"`
	TenantID string `json:"tenant_id"`
}

type CreateFloatingIPRequest struct {
	TenantID       string `json:"tenant_id"`
	ExternalIP     string `json:"external_ip"`
	InternalPortID string `json:"internal_port_id"`
	InternalIP     string `json:"internal_ip"`
}
