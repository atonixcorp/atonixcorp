package generic

import (
	"fmt"

	"atnetwork/core"
)

type StaticBackend struct{}

func NewStaticBackend() *StaticBackend {
	return &StaticBackend{}
}

// -----------------------------
// NETWORK
// -----------------------------

func (b *StaticBackend) ApplyNetwork(n core.Network) error {
	fmt.Printf("[GENERIC] ApplyNetwork: ID=%s Name=%s CIDR=%s Tenant=%s External=%v\n",
		n.ID, n.Name, n.CIDR, n.TenantID, n.External)

	// In real deployments:
	// - Write static route files
	// - Update /etc/network/interfaces
	// - Update simple VLAN configs
	// - Notify node agents

	return nil
}

// -----------------------------
// SUBNET
// -----------------------------

func (b *StaticBackend) ApplySubnet(s core.Subnet) error {
	fmt.Printf("[GENERIC] ApplySubnet: ID=%s NetworkID=%s CIDR=%s Gateway=%s\n",
		s.ID, s.NetworkID, s.CIDR, s.GatewayIP)

	return nil
}

// -----------------------------
// ROUTER
// -----------------------------

func (b *StaticBackend) ApplyRouter(r core.Router) error {
	fmt.Printf("[GENERIC] ApplyRouter: ID=%s Name=%s ExternalNetworkID=%s Tenant=%s\n",
		r.ID, r.Name, r.ExternalNetworkID, r.TenantID)

	return nil
}

// -----------------------------
// PORT
// -----------------------------

func (b *StaticBackend) ApplyPort(p core.Port) error {
	fmt.Printf("[GENERIC] ApplyPort: ID=%s NetworkID=%s DeviceID=%s MAC=%s\n",
		p.ID, p.NetworkID, p.DeviceID, p.MAC)

	return nil
}

// -----------------------------
// SECURITY GROUP
// -----------------------------

func (b *StaticBackend) ApplySecurityGroup(sg core.SecurityGroup) error {
	fmt.Printf("[GENERIC] ApplySecurityGroup: ID=%s Name=%s Tenant=%s\n",
		sg.ID, sg.Name, sg.TenantID)

	for _, rule := range sg.Rules {
		fmt.Printf("  Rule: %s %s %s %s\n",
			rule.Direction, rule.Protocol, rule.PortRange, rule.CIDR)
	}

	return nil
}

// -----------------------------
// OPTIONAL LIST METHODS (for reconciliation)
// -----------------------------

func (b *StaticBackend) ListNetworks() ([]core.Network, error) {
	return nil, nil
}

func (b *StaticBackend) ListSubnets() ([]core.Subnet, error) {
	return nil, nil
}

func (b *StaticBackend) ListRouters() ([]core.Router, error) {
	return nil, nil
}

func (b *StaticBackend) ListPorts() ([]core.Port, error) {
	return nil, nil
}

func (b *StaticBackend) ListSecurityGroups() ([]core.SecurityGroup, error) {
	return nil, nil
}
