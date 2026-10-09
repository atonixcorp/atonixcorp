package core

type Store interface {
	// Network
	CreateNetwork(Network) error
	GetNetwork(id string) (Network, error)

	// Subnet
	CreateSubnet(Subnet) error
	GetSubnet(id string) (Subnet, error)
	GetSubnetByNetwork(networkID string) (Subnet, error)

	// Router
	CreateRouter(Router) error
	GetRouter(id string) (Router, error)
	GetRouterBySubnet(subnetID string) (Router, error)

	// Router Interface
	CreateRouterInterface(RouterInterface) error

	// Port
	CreatePort(Port) error
	GetPort(id string) (Port, error)
	ListPorts() ([]Port, error) // <-- add this

	// Security Group
	CreateSecurityGroup(SecurityGroup) error

	// Floating IP
	CreateFloatingIP(FloatingIP) error

	// NAT
	CreateNATRule(NATRule) error
}
