package core

import (
	"fmt"
)

// Controller orchestrates all operations between the store and backends.
type Controller struct {
	store    Store
	backends []Backend
}

// NewController creates a new controller instance.
func NewController(store Store, backends []Backend) *Controller {
	return &Controller{
		store:    store,
		backends: backends,
	}
}

//
// NETWORK
//

func (c *Controller) CreateNetwork(n Network) error {
	if err := c.store.CreateNetwork(n); err != nil {
		return fmt.Errorf("store network: %w", err)
	}

	for _, b := range c.backends {
		if err := b.ApplyNetwork(n); err != nil {
			return fmt.Errorf("backend network apply failed: %w", err)
		}
	}

	return nil
}

//
// SUBNET
//

func (c *Controller) CreateSubnet(s Subnet) error {
	if err := c.store.CreateSubnet(s); err != nil {
		return fmt.Errorf("store subnet: %w", err)
	}

	for _, b := range c.backends {
		if err := b.ApplySubnet(s); err != nil {
			return fmt.Errorf("backend subnet apply failed: %w", err)
		}
	}

	return nil
}

//
// ROUTER
//

func (c *Controller) CreateRouter(r Router) error {
	if err := c.store.CreateRouter(r); err != nil {
		return fmt.Errorf("store router: %w", err)
	}

	for _, b := range c.backends {
		if rb, ok := b.(RouterBackend); ok {
			if err := rb.ApplyRouter(r); err != nil {
				return fmt.Errorf("backend router apply failed: %w", err)
			}
		}
	}

	return nil
}

//
// ROUTER INTERFACE
//

func (c *Controller) AttachRouterInterface(ri RouterInterface) error {
	if err := c.store.CreateRouterInterface(ri); err != nil {
		return fmt.Errorf("store router interface: %w", err)
	}

	router, err := c.store.GetRouter(ri.RouterID)
	if err != nil {
		return fmt.Errorf("get router: %w", err)
	}

	subnet, err := c.store.GetSubnet(ri.SubnetID)
	if err != nil {
		return fmt.Errorf("get subnet: %w", err)
	}

	for _, b := range c.backends {
		if rb, ok := b.(RouterBackend); ok {
			if err := rb.AttachInterface(router, subnet, ri); err != nil {
				return fmt.Errorf("backend attach interface failed: %w", err)
			}
		}
	}

	return nil
}

//
// PORT
//

func (c *Controller) CreatePort(p Port) error {
	if err := c.store.CreatePort(p); err != nil {
		return fmt.Errorf("store port: %w", err)
	}

	for _, b := range c.backends {
		if err := b.ApplyPort(p); err != nil {
			return fmt.Errorf("backend port apply failed: %w", err)
		}
	}

	return nil
}

//
// SECURITY GROUP
//

func (c *Controller) CreateSecurityGroup(sg SecurityGroup) error {
	if err := c.store.CreateSecurityGroup(sg); err != nil {
		return fmt.Errorf("store security group: %w", err)
	}

	for _, b := range c.backends {
		if err := b.ApplySecurityGroup(sg); err != nil {
			return fmt.Errorf("backend security group apply failed: %w", err)
		}
	}

	return nil
}

//
// FLOATING IP + NAT
//

func (c *Controller) CreateFloatingIP(fip FloatingIP) error {
	if err := c.store.CreateFloatingIP(fip); err != nil {
		return fmt.Errorf("store floating IP: %w", err)
	}

	nat := NATRule{
		ID:         GenerateID("nat"),
		RouterID:   c.routerForPort(fip.InternalPortID),
		ExternalIP: fip.ExternalIP,
		InternalIP: fip.InternalIP,
		Type:       "dnat",
	}

	if err := c.store.CreateNATRule(nat); err != nil {
		return fmt.Errorf("store NAT rule: %w", err)
	}

	for _, b := range c.backends {
		if nb, ok := b.(NATBackend); ok {
			if err := nb.ApplyNATRule(nat); err != nil {
				return fmt.Errorf("backend NAT apply failed: %w", err)
			}
		}
	}

	return nil
}

//
// INTERNAL: Resolve router for a port
//

func (c *Controller) routerForPort(portID string) string {
	port, err := c.store.GetPort(portID)
	if err != nil {
		return ""
	}

	subnet, err := c.store.GetSubnetByNetwork(port.NetworkID)
	if err != nil {
		return ""
	}

	router, err := c.store.GetRouterBySubnet(subnet.ID)
	if err != nil {
		return ""
	}

	return router.ID
}
