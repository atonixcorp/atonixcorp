package core

type Backend interface {
	ApplyNetwork(Network) error
	ApplySubnet(Subnet) error
	ApplyRouter(Router) error
	ApplyPort(Port) error
	ApplySecurityGroup(SecurityGroup) error

	ListNetworks() ([]Network, error)
	ListSubnets() ([]Subnet, error)
	ListRouters() ([]Router, error)
	ListPorts() ([]Port, error)
	ListSecurityGroups() ([]SecurityGroup, error)
}

type RouterBackend interface {
	ApplyRouter(Router) error
	AttachInterface(Router, Subnet, RouterInterface) error
}

type NATBackend interface {
	ApplyNATRule(NATRule) error
}
