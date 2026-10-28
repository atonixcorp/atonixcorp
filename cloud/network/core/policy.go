package core

import (
	"errors"
	"fmt"
	"net/netip"
)

var ErrCIDROverlap = errors.New("CIDR overlaps an existing allocation")

// Policy is the interface used by the API layer.
type Policy interface {
	ValidateNetwork(Network) error
	ValidateSubnet(Subnet) error
	ValidateSubnetInNetwork(Subnet, Network) error
	ValidateRouter(Router) error
	ValidateRouterInterface(RouterInterface) error
	ValidatePort(Port) error
	ValidateFloatingIP(FloatingIP) error
}

// policyEngine is the concrete implementation.
type policyEngine struct{}

func NewPolicy() Policy {
	return &policyEngine{}
}

func (p *policyEngine) ValidateNetwork(n Network) error {
	if n.Name == "" {
		return fmt.Errorf("network name cannot be empty")
	}
	if n.TenantID == "" || n.ProjectID == "" {
		return fmt.Errorf("network must belong to a tenant and project")
	}
	if _, err := parseCanonicalPrefix(n.CIDR); err != nil {
		return fmt.Errorf("invalid network CIDR: %w", err)
	}
	return nil
}

func (p *policyEngine) ValidateSubnet(s Subnet) error {
	if s.NetworkID == "" {
		return fmt.Errorf("subnet must belong to a network")
	}
	prefix, err := parseCanonicalPrefix(s.CIDR)
	if err != nil {
		return fmt.Errorf("invalid subnet CIDR: %w", err)
	}
	if s.GatewayIP != "" {
		gateway, err := netip.ParseAddr(s.GatewayIP)
		if err != nil || !prefix.Contains(gateway) {
			return fmt.Errorf("gateway IP must be a valid address inside the subnet")
		}
		if prefix.Bits() < prefix.Addr().BitLen() && gateway == lastAddress(prefix) {
			return fmt.Errorf("gateway IP cannot be the subnet broadcast address")
		}
	}
	return nil
}

func (p *policyEngine) ValidateSubnetInNetwork(s Subnet, n Network) error {
	if err := p.ValidateSubnet(s); err != nil {
		return err
	}
	networkPrefix, err := parseCanonicalPrefix(n.CIDR)
	if err != nil {
		return fmt.Errorf("invalid parent network CIDR: %w", err)
	}
	subnetPrefix, _ := parseCanonicalPrefix(s.CIDR)
	if networkPrefix.Addr().Is4() != subnetPrefix.Addr().Is4() ||
		subnetPrefix.Bits() < networkPrefix.Bits() ||
		!networkPrefix.Contains(subnetPrefix.Addr()) {
		return fmt.Errorf("subnet CIDR must be contained in the parent network")
	}
	return nil
}

func parseCanonicalPrefix(value string) (netip.Prefix, error) {
	prefix, err := netip.ParsePrefix(value)
	if err != nil {
		return netip.Prefix{}, err
	}
	if prefix != prefix.Masked() {
		return netip.Prefix{}, fmt.Errorf("CIDR must use its network address")
	}
	return prefix, nil
}

func lastAddress(prefix netip.Prefix) netip.Addr {
	bytes := prefix.Addr().AsSlice()
	last := append([]byte(nil), bytes...)
	for bit := prefix.Bits(); bit < len(last)*8; bit++ {
		last[bit/8] |= 1 << (7 - uint(bit%8))
	}
	if prefix.Addr().Is4() {
		var address [4]byte
		copy(address[:], last)
		return netip.AddrFrom4(address)
	}
	var address [16]byte
	copy(address[:], last)
	return netip.AddrFrom16(address)
}

func (p *policyEngine) ValidateRouter(r Router) error {
	if r.Name == "" {
		return fmt.Errorf("router name cannot be empty")
	}
	if r.TenantID == "" {
		return fmt.Errorf("router must have a tenant")
	}
	return nil
}

func (p *policyEngine) ValidateRouterInterface(ri RouterInterface) error {
	if ri.RouterID == "" {
		return fmt.Errorf("router interface must have router ID")
	}
	if ri.SubnetID == "" {
		return fmt.Errorf("router interface must have subnet ID")
	}
	if ri.IPAddress == "" {
		return fmt.Errorf("router interface must have IP address")
	}
	return nil
}

func (p *policyEngine) ValidatePort(pt Port) error {
	if pt.NetworkID == "" {
		return fmt.Errorf("port must belong to a network")
	}
	if pt.DeviceID == "" {
		return fmt.Errorf("port must have a device ID")
	}
	if pt.MAC == "" {
		return fmt.Errorf("port must have a MAC address")
	}
	return nil
}

func (p *policyEngine) ValidateFloatingIP(fip FloatingIP) error {
	if fip.ExternalIP == "" {
		return fmt.Errorf("floating IP must have external IP")
	}
	if fip.InternalIP == "" {
		return fmt.Errorf("floating IP must have internal IP")
	}
	if fip.InternalPortID == "" {
		return fmt.Errorf("floating IP must map to a port")
	}
	return nil
}
