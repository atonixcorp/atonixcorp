package core

import "fmt"

// Policy is the interface used by the API layer.
type Policy interface {
	ValidateNetwork(Network) error
	ValidateSubnet(Subnet) error
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
	if n.CIDR == "" {
		return fmt.Errorf("network CIDR cannot be empty")
	}
	if n.Name == "" {
		return fmt.Errorf("network name cannot be empty")
	}
	return nil
}

func (p *policyEngine) ValidateSubnet(s Subnet) error {
	if s.CIDR == "" {
		return fmt.Errorf("subnet CIDR cannot be empty")
	}
	if s.NetworkID == "" {
		return fmt.Errorf("subnet must belong to a network")
	}
	return nil
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
